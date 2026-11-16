import dayjs from "dayjs";
import { message } from "@/utils/message";
import type { PaginationProps } from "@pureadmin/table";
import { actTokenList, actTokenDel } from "@/api/sysmgr/sysset/systokenlist";
import { reactive, ref, onMounted, type Ref } from "vue";
import { useSysconfigStoreHook } from "@/store/modules/sysconfig";

/** 起止时间带上时区再发（例如 2026-11-15T00:00:00+08:00）：库里的时间带时区，不带的话后端不收 */
const zoned = (t: string) => (t ? dayjs(t).format() : "");

export function useTokenList(tableRef?: Ref) {
  const form = reactive({
    user_id: "",
    user_type: null as number | null,
    ip: "",
    starttime: "",
    endtime: ""
  });

  const dataList = ref([]);
  const loading = ref(true);
  const selectedNum = ref(0);
  const selectedIds = ref<number[]>([]);

  const pagination = reactive<PaginationProps>({
    total: 0,
    pageSize: useSysconfigStoreHook().pageSize,
    pageSizes: useSysconfigStoreHook().pageSizeOptions,
    currentPage: 1,
    background: true
  });

  const columns: TableColumnList = [
    {
      type: "selection",
      width: 55,
      reserveSelection: true,
      fixed: "left"
    },
    { label: "ID", prop: "id", width: 80 },
    { label: "用户ID", prop: "user_id", minWidth: 120 },
    {
      label: "用户类型",
      prop: "user_type",
      width: 110,
      cellRenderer: ({ row }) => {
        if (row.user_type === 1)
          return (
            <el-tag type="primary" size="small">
              管理员
            </el-tag>
          );
        if (row.user_type === 2)
          return (
            <el-tag type="success" size="small">
              用户
            </el-tag>
          );
        return (
          <el-tag type="info" size="small">
            {row.user_type}
          </el-tag>
        );
      }
    },
    { label: "IP地址", prop: "ip", minWidth: 140 },
    {
      label: "TokenID",
      prop: "token_id",
      minWidth: 280,
      showOverflowTooltip: true
    },
    {
      label: "登录时间",
      prop: "created_at",
      minWidth: 180,
      formatter: ({ created_at }) =>
        created_at ? dayjs(created_at).format("YYYY-MM-DD HH:mm:ss") : "-"
    },
    {
      label: "最后活跃",
      prop: "updated_at",
      minWidth: 180,
      formatter: ({ updated_at }) =>
        updated_at ? dayjs(updated_at).format("YYYY-MM-DD HH:mm:ss") : "-"
    },
    {
      label: "操作",
      fixed: "right",
      width: 80,
      slot: "operation"
    }
  ];

  async function onSearch() {
    loading.value = true;
    try {
      const res = await actTokenList({
        ...form,
        starttime: zoned(form.starttime),
        endtime: zoned(form.endtime),
        pageSize: pagination.pageSize,
        currentPage: pagination.currentPage
      });
      if (res.code === 0) {
        dataList.value = res.data?.list ?? [];
        pagination.total = res.data?.total ?? 0;
      } else {
        message(res.message || "获取在线用户列表失败", { type: "error" });
      }
    } finally {
      loading.value = false;
    }
  }

  function resetForm(formEl) {
    if (!formEl) return;
    formEl.resetFields();
    form.user_type = null;
    pagination.currentPage = 1;
    onSearch();
  }

  function handleSizeChange(val: number) {
    pagination.currentPage = 1;
    pagination.pageSize = val;
    onSearch();
  }

  function handleCurrentChange(val: number) {
    pagination.currentPage = val;
    onSearch();
  }

  function handleSelectionChange(rows: any[]) {
    selectedIds.value = rows.map(r => r.id);
    selectedNum.value = rows.length;
  }

  function onSelectionCancel() {
    selectedNum.value = 0;
    selectedIds.value = [];
    tableRef?.value?.getTableRef()?.clearSelection();
  }

  async function handleDelete(row) {
    const res = await actTokenDel({ ids: [row.id] }).catch(() => null);
    if (res?.code === 0) {
      message("强制下线成功", { type: "success" });
      onSearch();
    } else {
      message(res?.message || "操作失败", { type: "error" });
    }
  }

  async function onBatchDel() {
    if (!selectedIds.value.length) return;
    const res = await actTokenDel({ ids: selectedIds.value }).catch(() => null);
    if (res?.code === 0) {
      message("批量强制下线成功", { type: "success" });
      onSelectionCancel();
      onSearch();
    } else {
      message(res?.message || "操作失败", { type: "error" });
    }
  }

  onMounted(() => onSearch());

  return {
    form,
    loading,
    columns,
    dataList,
    pagination,
    selectedNum,
    onSearch,
    resetForm,
    handleDelete,
    handleSizeChange,
    handleCurrentChange,
    handleSelectionChange,
    onSelectionCancel,
    onBatchDel
  };
}
