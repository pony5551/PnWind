import editForm from "../form.vue";
// import { handleTree } from "@/utils/tree";
import { message } from "@/utils/message";
import { transformI18n } from "@/plugins/i18n";
import { addDialog } from "@/components/ReDialog";
import { reactive, ref, onMounted, h } from "vue";
import type { FormItemProps } from "../utils/types";
import { useRenderIcon } from "@/components/ReIcon/src/hooks";
import { cloneDeep, isAllEmpty, deviceDetection } from "@pureadmin/utils";
// import { id } from "element-plus/es/locale/index.mjs";
import {
  actMenuList,
  actMenuAdd,
  actMenuEdit,
  actMenuDel
} from "@/api/sysmgr/sysset/sysmenus";

export function useMenu() {
  const form = reactive({
    title: ""
  });

  const formRef = ref();
  const dataList = ref([]);
  const loading = ref(true);

  const getMenuType = (type, text = false) => {
    switch (type) {
      case 0:
        return text ? "菜单" : "primary";
      case 1:
        return text ? "iframe" : "warning";
      case 2:
        return text ? "外链" : "danger";
      case 3:
        return text ? "按钮" : "info";
    }
  };

  const columns: TableColumnList = [
    {
      label: "菜单名称",
      prop: "title",
      align: "left",
      cellRenderer: ({ row }) => (
        <>
          <span class="inline-block mr-1">
            {h(useRenderIcon(row.icon), {
              style: { paddingTop: "1px" }
            })}
          </span>
          <span>{transformI18n(row.title)}</span>
        </>
      )
    },
    {
      label: "菜单类型",
      prop: "menuType",
      width: 100,
      cellRenderer: ({ row, props }) => (
        <el-tag
          size={props.size}
          type={getMenuType(row.menuType)}
          effect="plain"
        >
          {getMenuType(row.menuType, true)}
        </el-tag>
      )
    },
    {
      label: "路由路径",
      prop: "path",
      align: "left"
    },
    {
      label: "组件路径",
      prop: "component",
      align: "left",
      formatter: ({ path, component }) =>
        isAllEmpty(component) ? path : component
    },
    {
      label: "权限标识",
      prop: "auths",
      align: "left"
    },
    {
      label: "排序",
      prop: "rank",
      width: 100
    },
    {
      label: "隐藏",
      prop: "showLink",
      formatter: ({ showLink }) => (showLink ? "否" : "是"),
      width: 100
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
    // console.log("onSearch", form);
    loading.value = true;
    const res = await actMenuList(); // 这里是返回一维数组结构，前端自行处理成树结构，返回格式要求：唯一id加父节点parentId，parentId取父节点id
    if (res.code === 0) {
      let newData = res.data;
      if (!isAllEmpty(form.title)) {
        // 前端搜索菜单名称
        newData = newData.filter(item =>
          transformI18n(item.title).includes(form.title)
        );
      }
      //dataList.value = handleTree(newData); // 处理成树结构
      dataList.value = newData; // 后端接口已处理成树结构，前端不需要再处理了
    } else {
      message(res.message || "获取菜单列表失败", {
        type: "error"
      });
    }

    setTimeout(() => {
      loading.value = false;
    }, 500);
  }

  function formatHigherMenuOptions(treeList) {
    if (!treeList || !treeList.length) return;
    const newTreeList = [];
    for (let i = 0; i < treeList.length; i++) {
      treeList[i].title = transformI18n(treeList[i].title);
      formatHigherMenuOptions(treeList[i].children);
      newTreeList.push(treeList[i]);
    }
    return newTreeList;
  }

  function openDialog(title = "新增", row?: FormItemProps) {
    addDialog({
      title: `${title}菜单`,
      props: {
        formInline: {
          id: row?.id ?? 0,
          menuType: row?.menuType ?? 0,
          higherMenuOptions: formatHigherMenuOptions(cloneDeep(dataList.value)),
          parentId: row?.parentId ?? 0,
          title: row?.title ?? "",
          name: row?.name ?? "",
          path: row?.path ?? "",
          component: row?.component ?? "",
          rank: row?.rank ?? 99,
          redirect: row?.redirect ?? "",
          icon: row?.icon ?? "",
          extraIcon: row?.extraIcon ?? "",
          enterTransition: row?.enterTransition ?? "",
          leaveTransition: row?.leaveTransition ?? "",
          activePath: row?.activePath ?? "",
          auths: row?.auths ?? "",
          frameSrc: row?.frameSrc ?? "",
          frameLoading: row?.frameLoading ?? true,
          keepAlive: row?.keepAlive ?? true,
          hiddenTag: row?.hiddenTag ?? false,
          fixedTag: row?.fixedTag ?? false,
          showLink: row?.showLink ?? true,
          showParent: row?.showParent ?? false,
          loged: row?.loged ?? false
        }
      },
      width: "45%",
      draggable: true,
      fullscreen: deviceDetection(),
      fullscreenIcon: true,
      closeOnClickModal: false,
      contentRenderer: () => h(editForm, { ref: formRef, formInline: null }),
      beforeSure: (done, { options }) => {
        const FormRef = formRef.value.getRef();
        const curData = options.props.formInline as FormItemProps;
        function chores() {
          // message(
          //   `您${title}了菜单名称为${transformI18n(curData.title)}的这条数据`,
          //   {
          //     type: "success"
          //   }
          // );
          done(); // 关闭弹框
          onSearch(); // 刷新表格数据
        }
        FormRef.validate(valid => {
          if (valid) {
            console.log("curData", curData);
            curData.higherMenuOptions = []; // 这里是为了不传这个字段，后端接口不需要这个字段
            // 表单规则校验通过
            if (title === "新增") {
              // 实际开发先调用新增接口，再进行下面操作
              actMenuAdd(curData).then(res => {
                if (res.code === 0) {
                  message(res.message || "操作成功", {
                    type: "success"
                  });
                  chores();
                } else {
                  message(res.message || "操作失败", {
                    type: "error"
                  });
                }
              });
            } else {
              // 实际开发先调用修改接口，再进行下面操作
              actMenuEdit(curData).then(res => {
                if (res.code === 0) {
                  message(res.message || "操作成功", {
                    type: "success"
                  });
                  chores();
                } else {
                  message(res.message || "操作失败", {
                    type: "error"
                  });
                }
              });
            }
          }
        });
      }
    });
  }

  // 递归收集当前节点及所有子孙节点的 id
  function collectIds(row: any): number[] {
    const ids: number[] = [row.id];
    if (row.children?.length) {
      for (const child of row.children) {
        ids.push(...collectIds(child));
      }
    }
    return ids;
  }

  async function handleDelete(row) {
    const ids = collectIds(row);
    const res = await actMenuDel({ ids }).catch(() => null);
    if (res?.code === 0) {
      message(
        `删除菜单【${transformI18n(row.title)}】成功${ids.length > 1 ? `，共删除 ${ids.length} 条记录` : ""}`,
        { type: "success" }
      );
      onSearch();
    } else {
      message(res?.message || "删除失败", { type: "error" });
    }
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
    /** 新增、修改菜单 */
    openDialog,
    /** 删除菜单 */
    handleDelete,
    handleSelectionChange
  };
}
