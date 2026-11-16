import dayjs from "dayjs";
import editForm from "../form.vue";
import resetPwdForm from "../resetpwd.vue";
import { message } from "@/utils/message";
import { addDialog } from "@/components/ReDialog";
import type { PaginationProps } from "@pureadmin/table";
import type { FormItemProps, ResetPwdFormProps, RoleOption } from "./types";
import { deviceDetection } from "@pureadmin/utils";
import {
  actUsersList,
  actUsersAdd,
  actUsersEdit,
  actUsersResetpwd,
  actUsersDel,
  actUsersRoleList
} from "@/api/sysmgr/sysuser/sysusers";
import { reactive, ref, computed, onMounted, h, type Ref } from "vue";
import md5 from "md5";
import { useSysconfigStoreHook } from "@/store/modules/sysconfig";
import { useUserStoreHook } from "@/store/modules/user";

// 用户类型映射
export const USERS_TYPE_MAP: Record<number, string> = {
  1: "个人",
  2: "个体户",
  3: "企业"
};

// 用户状态映射（和 db_sys_users.status 的取值一致）
export const USERS_STATUS_MAP: Record<number, { label: string; type: string }> =
  {
    0: { label: "待审核", type: "info" },
    1: { label: "正常", type: "success" },
    2: { label: "冻结", type: "warning" },
    3: { label: "注销", type: "danger" }
  };

export function useUsers(tableRef?: Ref) {
  const form = reactive({
    user_id: "",
    name: "",
    email: "",
    phone: "",
    type: 0,
    status: null as number | null
  });

  const formRef = ref();
  const dataList = ref([]);
  const loading = ref(true);
  const selectedNum = ref(0);
  const selectedIds = ref<number[]>([]);
  const roleOptions = ref<RoleOption[]>([]);

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
    { label: "ID", prop: "id", width: 80, hide: true },
    {
      label: "用户编号",
      prop: "user_id",
      width: 120
    },
    {
      label: "用户名称",
      prop: "name",
      minWidth: 160
    },
    {
      label: "用户类型",
      prop: "type",
      width: 110,
      cellRenderer: ({ row }) => USERS_TYPE_MAP[row.type] ?? row.type
    },
    {
      label: "状态",
      prop: "status",
      width: 90,
      cellRenderer: ({ row }) => {
        const s = USERS_STATUS_MAP[row.status];
        return s ? (
          <el-tag type={s.type as any} size="small">
            {s.label}
          </el-tag>
        ) : (
          row.status
        );
      }
    },
    {
      label: "邮箱",
      prop: "email",
      minWidth: 180,
      showOverflowTooltip: true,
      hide: true
    },
    {
      label: "用户手机",
      prop: "phone",
      width: 130
    },
    {
      label: "用户角色",
      prop: "roles",
      minWidth: 68,
      cellRenderer: ({ row }) => {
        const roles: string[] = row.roles || [];
        return (
          <div class="flex flex-wrap gap-1">
            {roles.map((code: string) => {
              const r = roleOptions.value.find(o => o.code === code);
              return (
                <el-tag key={code} size="small" type="info">
                  {r ? r.name : code}
                </el-tag>
              );
            })}
          </div>
        );
      }
    },
    {
      label: "现金余额",
      prop: "cash_balance",
      width: 120
    },
    {
      label: "资金余额",
      prop: "funds_balance",
      width: 120
    },
    {
      // verified_id 是实名信息的 id，0 表示还没有实名
      label: "已认证",
      prop: "verified_id",
      width: 90,
      cellRenderer: ({ row }) =>
        row.verified_id > 0 ? (
          <el-tag type="success" size="small">
            已认证
          </el-tag>
        ) : (
          <el-tag type="info" size="small">
            未认证
          </el-tag>
        )
    },
    {
      label: "注册时间",
      prop: "created_at",
      minWidth: 180,
      formatter: ({ created_at }) =>
        created_at ? dayjs(created_at).format("YYYY-MM-DD HH:mm:ss") : "-"
    },
    {
      label: "操作",
      fixed: "right",
      width: 200,
      slot: "operation"
    }
  ];

  async function onSearch() {
    loading.value = true;
    try {
      const res = await actUsersList({
        ...form,
        pageSize: pagination.pageSize,
        currentPage: pagination.currentPage
      });
      if (res.code === 0) {
        dataList.value = res.data?.list ?? [];
        pagination.total = res.data?.total ?? 0;
        pagination.pageSize = res.data?.pageSize ?? pagination.pageSize;
        pagination.currentPage =
          res.data?.currentPage ?? pagination.currentPage;
      } else {
        message(res.message || "获取用户列表失败", { type: "error" });
      }
    } finally {
      loading.value = false;
    }
  }

  function resetForm(formEl) {
    if (!formEl) return;
    formEl.resetFields();
    onSearch();
  }

  function handleSizeChange(val: number) {
    pagination.pageSize = val;
    onSearch();
  }

  function handleCurrentChange(val: number) {
    pagination.currentPage = val;
    onSearch();
  }

  async function handleDelete(row) {
    const res = await actUsersDel({ ids: [row.id] }).catch(() => null);
    if (res?.code === 0) {
      message("删除成功", { type: "success" });
      onSearch();
    } else {
      message(res?.message || "删除失败", { type: "error" });
    }
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
    const res = await actUsersDel({ ids: selectedIds.value }).catch(() => null);
    if (res?.code === 0) {
      message("批量删除成功", { type: "success" });
      onSelectionCancel();
      onSearch();
    } else {
      message(res?.message || "批量删除失败", { type: "error" });
    }
  }

  function openDialog(title = "新增", row?: FormItemProps) {
    addDialog({
      title: `${title}用户`,
      props: {
        formInline: {
          id: row?.id ?? 0,
          name: row?.name ?? "",
          type: row?.type ?? 0,
          status: row?.status ?? 0,
          email: row?.email ?? "",
          phone: row?.phone ?? "",
          roles: row?.roles ? [...row.roles] : [],
          roleOptions: roleOptions.value,
          config: row?.config ?? "",
          remark: row?.remark ?? ""
        } as FormItemProps
      },
      width: "560px",
      draggable: true,
      fullscreen: deviceDetection(),
      fullscreenIcon: true,
      closeOnClickModal: false,
      contentRenderer: () => h(editForm, { ref: formRef, formInline: null }),
      beforeSure: (done, { options }) => {
        const FormRef = formRef.value?.getRef();
        const curData = options.props.formInline as FormItemProps;
        FormRef?.validate(async valid => {
          if (!valid) return;
          // 配置留空时后端按 {} 存
          const payload = {
            ...curData,
            config: curData.config?.trim() || "",
            remark: curData.remark?.trim() || ""
          };
          const res =
            title === "新增"
              ? await actUsersAdd(payload).catch(() => null)
              : await actUsersEdit(payload).catch(() => null);
          if (res?.code === 0) {
            message(`${title}用户成功`, { type: "success" });
            done();
            onSearch();
          } else {
            message(res?.message || `${title}失败`, { type: "error" });
          }
        });
      }
    });
  }

  function openResetPwdDialog(row) {
    const resetFormRef = ref();
    addDialog({
      title: `重置用户密码【${row.name}】`,
      props: {
        formInline: {
          id: row.id,
          password: "",
          pay_password: ""
        } as ResetPwdFormProps
      },
      width: "460px",
      draggable: true,
      fullscreen: deviceDetection(),
      fullscreenIcon: true,
      closeOnClickModal: false,
      contentRenderer: () =>
        h(resetPwdForm, { ref: resetFormRef, formInline: null }),
      beforeSure: (done, { options }) => {
        const FormRef = resetFormRef.value?.getRef();
        const curData = options.props.formInline as ResetPwdFormProps;
        FormRef?.validate(async valid => {
          if (!valid) return;
          if (!curData.password && !curData.pay_password) {
            message("请至少填写一个密码", { type: "warning" });
            return;
          }
          const payload: ResetPwdFormProps = {
            id: curData.id,
            password: "",
            pay_password: ""
          };
          if (curData.password) {
            // 登录密码：3次 MD5
            payload.password = md5(md5(md5(curData.password)));
          }
          if (curData.pay_password) {
            // 资金密码：5次 MD5
            payload.pay_password = md5(
              md5(md5(md5(md5(curData.pay_password))))
            );
          }
          const res = await actUsersResetpwd(payload).catch(() => null);
          if (res?.code === 0) {
            message("密码重置成功", { type: "success" });
            done();
          } else {
            message(res?.message || "密码重置失败", { type: "error" });
          }
        });
      }
    });
  }

  onMounted(async () => {
    try {
      const res = await actUsersRoleList({ parent_id: 2 });
      if (res.code === 0) roleOptions.value = res.data ?? [];
    } catch (e) {
      console.error(e);
    }
    onSearch();
  });

  return {
    form,
    loading,
    columns,
    dataList,
    pagination,
    selectedNum,
    isSuperAdmin,
    roleOptions,
    onSearch,
    resetForm,
    openDialog,
    openResetPwdDialog,
    handleDelete,
    handleSizeChange,
    handleCurrentChange,
    handleSelectionChange,
    onSelectionCancel,
    onbatchDel
  };
}
