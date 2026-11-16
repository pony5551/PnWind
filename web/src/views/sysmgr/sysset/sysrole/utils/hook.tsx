import dayjs from "dayjs";
import editForm from "../form.vue";
import { handleTree } from "@/utils/tree";
import { message } from "@/utils/message";
import { ElMessageBox } from "element-plus";
import { usePublicHooks } from "../../hooks";
import { transformI18n } from "@/plugins/i18n";
import { addDialog } from "@/components/ReDialog";
import type { FormItemProps } from "../utils/types";
import { getKeyList, deviceDetection, cloneDeep } from "@pureadmin/utils";
import { actRoleMenuList } from "@/api/sysmgr/sysset/sysmenus";
import {
  actRoleList,
  actRoleAdd,
  actRoleEdit,
  actMenuSetGet,
  actMenuSetSave
} from "@/api/sysmgr/sysset/sysrole";
import { type Ref, reactive, ref, onMounted, h, watch } from "vue";

export function useRole(treeRef: Ref) {
  const form = reactive({
    name: "",
    code: "",
    status: ""
  });
  const curRow = ref();
  const formRef = ref();
  const dataList = ref([]);
  const treeIds = ref([]);
  const treeData = ref([]);
  const isShow = ref(false);
  const loading = ref(true);
  const isLinkage = ref(true);
  const treeSearchValue = ref();
  const switchLoadMap = ref({});
  const isExpandAll = ref(false);
  const isSelectAll = ref(false);
  const { switchStyle } = usePublicHooks();
  const treeProps = {
    value: "name",
    label: "title",
    children: "children"
  };

  const columns: TableColumnList = [
    {
      label: "角色名称",
      prop: "name",
      width: 200,
      align: "left"
    },
    {
      label: "角色标识",
      prop: "code",
      width: 180,
      align: "left"
    },
    {
      label: "排序",
      prop: "sort",
      width: 70,
      align: "left"
    },
    {
      label: "状态",
      cellRenderer: scope => (
        <el-switch
          size={scope.props.size === "small" ? "small" : "default"}
          loading={switchLoadMap.value[scope.index]?.loading}
          v-model={scope.row.status}
          active-value={1}
          inactive-value={0}
          active-text="已启用"
          inactive-text="已停用"
          inline-prompt
          style={switchStyle.value}
          onChange={() => onChange(scope as any)}
        />
      ),
      minWidth: 90,
      width: 90
    },
    {
      label: "备注",
      prop: "remark",
      minWidth: 160
    },
    {
      label: "创建时间",
      prop: "created_at",
      minWidth: 160,
      width: 190,
      formatter: ({ created_at }) =>
        dayjs(created_at).format("YYYY-MM-DD HH:mm:ss")
    },
    {
      label: "操作",
      fixed: "right",
      width: 210,
      slot: "operation"
    }
  ];

  function onChange({ row, index }) {
    ElMessageBox.confirm(
      `确认要<strong>${
        row.status === 0 ? "停用" : "启用"
      }</strong><strong style='color:var(--el-color-primary)'>${
        row.name
      }</strong>吗?`,
      "系统提示",
      {
        confirmButtonText: "确定",
        cancelButtonText: "取消",
        type: "warning",
        dangerouslyUseHTMLString: true,
        draggable: true
      }
    )
      .then(() => {
        switchLoadMap.value[index] = Object.assign(
          {},
          switchLoadMap.value[index],
          { loading: true }
        );
        actRoleEdit({
          id: row.id,
          parent_id: row.parent_id ?? 0,
          name: row.name,
          code: row.code,
          sort: row.sort,
          status: row.status,
          remark: row.remark ?? ""
        }).then(res => {
          switchLoadMap.value[index] = Object.assign(
            {},
            switchLoadMap.value[index],
            { loading: false }
          );
          if (res.code === 0) {
            message(`已${row.status === 0 ? "停用" : "启用"}${row.name}`, {
              type: "success"
            });
          } else {
            message(res.message || "操作失败", { type: "error" });
            row.status === 0 ? (row.status = 1) : (row.status = 0);
          }
        });
      })
      .catch(() => {
        row.status === 0 ? (row.status = 1) : (row.status = 0);
      });
  }

  function handleDelete(row) {
    message(`您删除了角色名称为${row.name}的这条数据`, { type: "success" });
    onSearch();
  }

  async function onSearch() {
    loading.value = true;
    const res = await actRoleList({ ...form });
    if (res.code === 0) {
      // 后端返回平铺数组，handleTree 按 id/parent_id 构建树
      dataList.value = handleTree(res.data, "id", "parent_id");
    } else {
      message(res.message || "获取角色列表失败", { type: "error" });
    }
    setTimeout(() => {
      loading.value = false;
    }, 500);
  }

  const resetForm = formEl => {
    if (!formEl) return;
    formEl.resetFields();
    onSearch();
  };

  /** 将已停用角色标记为 disabled，用于上级角色级联选择器 */
  function formatHigherRoleOptions(treeList) {
    if (!treeList || !treeList.length) return [];
    return treeList.map(item => {
      const node = { ...item, disabled: item.status === 0 };
      if (node.children?.length) {
        node.children = formatHigherRoleOptions(node.children);
      }
      return node;
    });
  }

  function openDialog(title = "新增", row?: FormItemProps) {
    addDialog({
      title: `${title}角色`,
      props: {
        formInline: {
          higherRoleOptions: formatHigherRoleOptions(cloneDeep(dataList.value)),
          id: row?.id ?? 0,
          parent_id: row?.parent_id ?? 0,
          name: row?.name ?? "",
          code: row?.code ?? "",
          sort: row?.sort ?? 99,
          status: row?.status ?? 1,
          remark: row?.remark ?? ""
        }
      },
      width: "40%",
      draggable: true,
      fullscreen: deviceDetection(),
      fullscreenIcon: true,
      closeOnClickModal: false,
      contentRenderer: () => h(editForm, { ref: formRef, formInline: null }),
      beforeSure: (done, { options }) => {
        const FormRef = formRef.value.getRef();
        const curData = options.props.formInline as FormItemProps;
        function chores() {
          done();
          onSearch();
        }
        FormRef.validate(valid => {
          if (valid) {
            if (title === "新增") {
              actRoleAdd(curData).then(res => {
                if (res.code === 0) {
                  message(res.message || "操作成功", { type: "success" });
                  chores();
                } else {
                  message(res.message || "操作失败", { type: "error" });
                }
              });
            } else {
              actRoleEdit(curData).then(res => {
                if (res.code === 0) {
                  message(res.message || "操作成功", { type: "success" });
                  chores();
                } else {
                  message(res.message || "操作失败", { type: "error" });
                }
              });
            }
          }
        });
      }
    });
  }

  /** 菜单权限 */
  async function handleMenu(row?: any) {
    const { id } = row;
    if (id) {
      curRow.value = row;
      isShow.value = true;
      const { code, data } = await actMenuSetGet({ id });
      if (code === 0) {
        treeRef.value.setCheckedKeys(data);
      }
    } else {
      curRow.value = null;
      isShow.value = false;
    }
  }

  /** 高亮当前权限选中行 */
  function rowStyle({ row: { id } }) {
    return {
      cursor: "pointer",
      background: id === curRow.value?.id ? "var(--el-fill-color-light)" : ""
    };
  }

  /** 菜单权限-保存 */
  function handleSave() {
    const { id, name } = curRow.value;
    actMenuSetSave({
      id,
      menuset: treeRef.value.getCheckedKeys()
    }).then(res => {
      if (res.code === 0) {
        message(res.message || `${name}操作成功`, { type: "success" });
      } else {
        message(res.message || `${name}操作失败`, { type: "error" });
      }
    });
  }

  const onQueryChanged = (query: string) => {
    treeRef.value?.filter(query);
  };

  const filterMethod = (query: string, node) => {
    return transformI18n(node.title)?.includes(query);
  };

  onMounted(async () => {
    onSearch();
    const { code, data } = await actRoleMenuList();
    if (code === 0) {
      treeIds.value = getKeyList(data, "name");
      treeData.value = data;
    }
  });

  watch(isExpandAll, val => {
    val
      ? treeRef.value.setExpandedKeys(treeIds.value)
      : treeRef.value.setExpandedKeys([]);
  });

  watch(isSelectAll, val => {
    val
      ? treeRef.value.setCheckedKeys(treeIds.value)
      : treeRef.value.setCheckedKeys([]);
  });

  function handleSelectionChange(val) {
    console.log("selection:", val);
  }

  return {
    form,
    isShow,
    curRow,
    loading,
    columns,
    rowStyle,
    dataList,
    treeData,
    treeProps,
    isLinkage,
    isExpandAll,
    isSelectAll,
    treeSearchValue,
    onSearch,
    resetForm,
    openDialog,
    handleMenu,
    handleSave,
    handleDelete,
    filterMethod,
    transformI18n,
    onQueryChanged,
    handleSelectionChange
  };
}
