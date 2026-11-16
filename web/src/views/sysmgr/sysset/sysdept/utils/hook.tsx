import dayjs from "dayjs";
import editForm from "../form.vue";
import { handleTree } from "@/utils/tree";
import { message } from "@/utils/message";
import { usePublicHooks } from "../../hooks";
import { addDialog } from "@/components/ReDialog";
import { reactive, ref, onMounted, h } from "vue";
import type { FormItemProps } from "../utils/types";
import { cloneDeep, deviceDetection } from "@pureadmin/utils";
import {
  actDeptList,
  actDeptAdd,
  actDeptEdit,
  actDeptDel
} from "@/api/sysmgr/sysset/sysdept";

export function useDept() {
  const form = reactive({
    name: "",
    status: ""
  });

  const formRef = ref();
  const dataList = ref([]);
  const loading = ref(true);
  const { tagStyle } = usePublicHooks();

  const columns: TableColumnList = [
    {
      label: "部门名称",
      prop: "name",
      width: 180,
      align: "left"
    },
    {
      label: "排序",
      prop: "sort",
      minWidth: 70
    },
    {
      label: "状态",
      prop: "status",
      minWidth: 100,
      cellRenderer: ({ row, props }) => (
        <el-tag size={props.size} style={tagStyle.value(row.status)}>
          {row.status === 1 ? "启用" : "停用"}
        </el-tag>
      )
    },
    {
      label: "创建时间",
      minWidth: 200,
      prop: "created_at",
      formatter: ({ created_at }) =>
        dayjs(created_at).format("YYYY-MM-DD HH:mm:ss")
    },
    {
      label: "备注",
      prop: "remark",
      minWidth: 320
    },
    {
      label: "操作",
      fixed: "right",
      width: 210,
      slot: "operation"
    }
  ];

  function handleSelectionChange(val) {
    console.log("handleSelectionChange", val);
  }

  function resetForm(formEl) {
    if (!formEl) return;
    formEl.resetFields();
    onSearch();
  }

  async function onSearch() {
    loading.value = true;
    const fetchDataContext = reactive({
      requestParams: { ...form }
    });
    console.log("fetchDataContext", fetchDataContext);
    const res = await actDeptList(fetchDataContext.requestParams);
    if (res.code === 0) {
      dataList.value = handleTree(res.data, "id", "parent_id"); // 处理成树结构，依赖 id / parent_id
    } else {
      message(res.message || "获取部门列表失败", { type: "error" });
    }
    setTimeout(() => {
      loading.value = false;
    }, 500);
  }

  function formatHigherDeptOptions(treeList) {
    // 将已停用部门标记为 disabled，用于上级部门级联选择器
    if (!treeList || !treeList.length) return;
    const newTreeList = [];
    for (let i = 0; i < treeList.length; i++) {
      treeList[i].disabled = treeList[i].status === 0 ? true : false;
      formatHigherDeptOptions(treeList[i].children);
      newTreeList.push(treeList[i]);
    }
    return newTreeList;
  }

  function openDialog(title = "新增", row?: FormItemProps) {
    addDialog({
      title: `${title}部门`,
      props: {
        formInline: {
          higherDeptOptions: formatHigherDeptOptions(cloneDeep(dataList.value)),
          id: row?.id ?? 0,
          parent_id: row?.parent_id ?? 0,
          name: row?.name ?? "",
          principal: row?.principal ?? "",
          phone: row?.phone ?? "",
          email: row?.email ?? "",
          sort: row?.sort ?? 0,
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
              actDeptAdd(curData).then(res => {
                if (res.code === 0) {
                  message(res.message || "新增部门成功", { type: "success" });
                  chores();
                } else {
                  message(res.message || "新增失败", { type: "error" });
                }
              });
            } else {
              actDeptEdit(curData).then(res => {
                if (res.code === 0) {
                  message(res.message || "修改部门成功", { type: "success" });
                  chores();
                } else {
                  message(res.message || "修改失败", { type: "error" });
                }
              });
            }
          }
        });
      }
    });
  }

  function handleDelete(row) {
    actDeptDel({ ids: [row.id] }).then(res => {
      if (res.code === 0) {
        message("删除部门成功", { type: "success" });
        onSearch();
      } else {
        message(res.message || "删除失败", { type: "error" });
      }
    });
  }

  onMounted(() => {
    onSearch();
  });

  return {
    form,
    loading,
    columns,
    dataList,
    /** 搜索 */
    onSearch,
    /** 重置 */
    resetForm,
    /** 新增、修改部门 */
    openDialog,
    /** 删除部门 */
    handleDelete,
    handleSelectionChange
  };
}
