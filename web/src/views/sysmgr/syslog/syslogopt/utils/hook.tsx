import dayjs from "dayjs";
import { message } from "@/utils/message";
import type { Ref } from "vue";
import type { PaginationProps } from "@pureadmin/table";
import {
  actLogOptList,
  actLogOptDel,
  actLogOptClear
} from "@/api/sysmgr/syslog/syslogopt";
import { ElMessageBox } from "element-plus";
import { reactive, ref, computed, onMounted } from "vue";
import { useSysconfigStoreHook } from "@/store/modules/sysconfig";
import { useUserStoreHook } from "@/store/modules/user";

function todayRange(): [string, string] {
  const pad = (n: number) => String(n).padStart(2, "0");
  const d = new Date();
  const date = `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
  return [`${date} 00:00:00`, `${date} 23:59:59`];
}

function startOfDay(d: Date): Date {
  const x = new Date(d);
  x.setHours(0, 0, 0, 0);
  return x;
}
function endOfDay(d: Date): Date {
  const x = new Date(d);
  x.setHours(23, 59, 59, 0);
  return x;
}

export const dateShortcuts = [
  {
    text: "昨天",
    value: () => {
      const d = new Date();
      d.setDate(d.getDate() - 1);
      return [startOfDay(d), endOfDay(d)] as [Date, Date];
    }
  },
  {
    text: "今天",
    value: () => {
      const d = new Date();
      return [startOfDay(d), endOfDay(d)] as [Date, Date];
    }
  },
  {
    text: "三天前",
    value: () => {
      const start = new Date();
      start.setDate(start.getDate() - 3);
      const end = new Date();
      return [startOfDay(start), endOfDay(end)] as [Date, Date];
    }
  },
  {
    text: "上月",
    value: () => {
      const now = new Date();
      const start = new Date(now.getFullYear(), now.getMonth() - 1, 1);
      const end = new Date(now.getFullYear(), now.getMonth(), 0);
      return [startOfDay(start), endOfDay(end)] as [Date, Date];
    }
  },
  {
    text: "本月",
    value: () => {
      const now = new Date();
      const start = new Date(now.getFullYear(), now.getMonth(), 1);
      const end = new Date(now.getFullYear(), now.getMonth() + 1, 0);
      return [startOfDay(start), endOfDay(end)] as [Date, Date];
    }
  }
];

export const optTypeOptions = [
  { value: "用户操作", label: "用户操作" },
  { value: "系统操作", label: "系统操作" },
  { value: "登录", label: "登录" }
];

/** 用户类型：和操作日志表 user_type 的取值一样 */
export const userTypeOptions = [
  { value: 1, label: "管理员" },
  { value: 2, label: "用户" }
];

export function useLogOpt(tableRef?: Ref) {
  const form = reactive({
    user_id: "",
    user_type: null as number | null,
    type: "用户操作",
    skey: "",
    timeRange: todayRange() as string[] | null
  });

  const dataList = ref([]);
  const loading = ref(true);
  const selectedNum = ref(0);
  const selectedIds = ref<number[]>([]);
  const isSuperAdmin = computed(
    () => useUserStoreHook().userid === "superadmin"
  );

  const pagination = reactive<PaginationProps>({
    total: 0,
    pageSize: useSysconfigStoreHook().pageSize,
    pageSizes: useSysconfigStoreHook().pageSizeOptions,
    currentPage: 1,
    background: true
  });

  const columns: TableColumnList = [
    { type: "selection", width: 55, align: "left" },
    { label: "id", prop: "id", width: 100 },
    { label: "用户", prop: "user_id", minWidth: 110 },
    {
      label: "用户类型",
      prop: "user_type",
      minWidth: 100,
      formatter: ({ user_type }) =>
        userTypeOptions.find(o => o.value === user_type)?.label ?? user_type
    },
    { label: "操作类型", prop: "type", minWidth: 110 },
    { label: "事件", prop: "events", minWidth: 120 },
    { label: "ip", prop: "ip", minWidth: 160 },
    {
      label: "地址",
      prop: "page_path",
      minWidth: 220,
      showOverflowTooltip: true
    },
    {
      label: "form参数",
      prop: "page_form",
      minWidth: 220,
      showOverflowTooltip: true
    },
    {
      label: "query参数",
      prop: "page_query",
      minWidth: 160,
      showOverflowTooltip: true
    },
    {
      label: "详情",
      prop: "details",
      minWidth: 180,
      showOverflowTooltip: true
    },
    {
      label: "操作时间",
      prop: "created_at",
      minWidth: 170,
      formatter: ({ created_at }) =>
        created_at ? dayjs(created_at).format("YYYY-MM-DD HH:mm:ss") : "-"
    }
  ];

  async function onSearch() {
    loading.value = true;
    try {
      // 起止时间带上时区再发（例如 2026-11-15T00:00:00+08:00）：库里的时间带时区，不带的话后端不收
      const [starttime = "", endtime = ""] = (form.timeRange ?? []).map(t =>
        dayjs(t).format()
      );
      const res = await actLogOptList({
        user_id: form.user_id,
        user_type: form.user_type,
        type: form.type,
        skey: form.skey,
        starttime,
        endtime,
        pageSize: pagination.pageSize,
        currentPage: pagination.currentPage
      });
      if (res.code === 0) {
        dataList.value = res.data?.list ?? [];
        pagination.total = res.data?.total ?? 0;
      } else {
        message(res.message || "获取操作日志失败", { type: "error" });
      }
    } finally {
      loading.value = false;
    }
  }

  function resetForm(formEl) {
    if (formEl) formEl.resetFields();
    form.user_id = "";
    form.user_type = null;
    form.type = "用户操作";
    form.skey = "";
    form.timeRange = todayRange();
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

  async function onbatchDel() {
    if (!selectedIds.value.length) return;
    const res = await actLogOptDel({ ids: selectedIds.value }).catch(
      () => null
    );
    if (res?.code === 0) {
      message("批量删除成功", { type: "success" });
      onSelectionCancel();
      onSearch();
    } else {
      message(res?.message || "批量删除失败", { type: "error" });
    }
  }

  function onClear() {
    if (!isSuperAdmin.value) {
      message("无权操作，仅限超级管理员", { type: "error" });
      return;
    }
    ElMessageBox.confirm(
      "确认要清空所有操作日志吗？此操作不可恢复！",
      "系统提示",
      {
        confirmButtonText: "确定清空",
        cancelButtonText: "取消",
        type: "warning"
      }
    )
      .then(async () => {
        const res = await actLogOptClear();
        if (res.code === 0) {
          message(res.message || "操作日志清空成功", { type: "success" });
          pagination.currentPage = 1;
          onSelectionCancel();
          onSearch();
        } else {
          message(res.message || "清空失败", { type: "error" });
        }
      })
      .catch(() => {});
  }

  onMounted(() => onSearch());

  return {
    form,
    loading,
    columns,
    dataList,
    pagination,
    selectedNum,
    isSuperAdmin,
    userTypeOptions,
    optTypeOptions,
    dateShortcuts,
    onSearch,
    resetForm,
    handleSizeChange,
    handleCurrentChange,
    handleSelectionChange,
    onSelectionCancel,
    onbatchDel,
    onClear
  };
}
