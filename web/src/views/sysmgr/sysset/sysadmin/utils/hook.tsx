import "./reset.css";
import dayjs from "dayjs";
import md5 from "md5";
import roleForm from "../form/role.vue";
import editForm from "../form/index.vue";
import { zxcvbn } from "@zxcvbn-ts/core";
import { handleTree } from "@/utils/tree";
import { message } from "@/utils/message";
import { addDialog } from "@/components/ReDialog";
import { usePublicHooks } from "../../hooks";
import type { PaginationProps } from "@pureadmin/table";
import ReCropperPreview from "@/components/ReCropperPreview";
import type { FormItemProps, RoleFormItemProps, RoleOption } from "./types";
import { getKeyList, isAllEmpty, deviceDetection } from "@pureadmin/utils";
import {
  actAdminList,
  actAdminAdd,
  actAdminEdit,
  actAdminDel,
  actAdminSwitch,
  actAdminResetPwd,
  actAdminSetAvatar,
  actAdminRoleList,
  actAdminRoleSave,
  actAdminDeptList
} from "@/api/sysmgr/sysset/sysadmin";
import {
  ElForm,
  ElInput,
  ElFormItem,
  ElProgress,
  ElMessageBox
} from "element-plus";
import { type Ref, h, ref, reactive, computed, watch, onMounted } from "vue";
import { useSysconfigStoreHook } from "@/store/modules/sysconfig";
import { useUserStoreHook } from "@/store/modules/user";

export function useAdmin(tableRef: Ref, treeRef: Ref) {
  const form = reactive({
    username: "",
    phone: "",
    status: "",
    dept_id: 0
  });
  const formRef = ref();
  const ruleFormRef = ref();
  const dataList = ref([]);
  const loading = ref(true);
  const avatarInfo = ref<any>();
  const switchLoadMap = ref({});
  const { switchStyle } = usePublicHooks();
  // 超级管理员可见已删除用户（后端对 superadmin 不过滤 deleted_at）
  const isSuperAdmin = computed(() => useUserStoreHook().userid === "superadmin");
  const treeData = ref([]);
  const treeLoading = ref(true);
  const selectedNum = ref(0);
  const roleOptions = ref<RoleOption[]>([]);
  const deptOptions = ref<any[]>([]);

  const pagination = reactive<PaginationProps>({
    total: 0,
    pageSize: useSysconfigStoreHook().pageSize,
    pageSizes: useSysconfigStoreHook().pageSizeOptions,
    currentPage: 1,
    background: true
  });

  const columns: TableColumnList = [
    {
      label: "勾选列",
      type: "selection",
      fixed: "left",
      reserveSelection: true
    },
    { label: "ID", prop: "id", width: 80 },
    {
      label: "头像",
      prop: "avatar",
      width: 80,
      cellRenderer: ({ row }) =>
        row.avatar ? (
          <el-image
            fit="cover"
            preview-teleported={true}
            src={row.avatar}
            preview-src-list={[row.avatar]}
            class="size-8 rounded-full align-middle"
          />
        ) : (
          <el-avatar size={32} icon="UserFilled" />
        )
    },
    {
      label: "用户名",
      prop: "username",
      minWidth: 150,
      cellRenderer: ({ row }) => (
        <div>
          <div>{row.username}</div>
          <div class="text-xs text-gray-400">{row.user_id}</div>
        </div>
      )
    },
    { label: "真实姓名", prop: "real_name", minWidth: 100 },
    {
      label: "性别",
      prop: "gender",
      width: 70,
      cellRenderer: ({ row }) => (
        <span>{row.gender === 0 ? "男" : row.gender === 1 ? "女" : "-"}</span>
      )
    },
    {
      label: "部门",
      prop: "dept_id",
      minWidth: 120,
      cellRenderer: ({ row }) => {
        if (!row.dept_id) return <span>-</span>;
        const dept = deptOptions.value.find(d => d.id === row.dept_id);
        return <span>{dept ? dept.name : row.dept_id}</span>;
      }
    },
    { label: "手机号", prop: "phone", minWidth: 130 },
    { label: "邮箱", prop: "email", minWidth: 160 },
    {
      label: "角色",
      prop: "roles",
      minWidth: 160,
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
      label: "状态",
      prop: "status",
      width: 110,
      cellRenderer: scope => (
        <el-switch
          size={scope.props.size === "small" ? "small" : "default"}
          loading={switchLoadMap.value[scope.index]?.loading}
          v-model={scope.row.status}
          active-value={1}
          inactive-value={0}
          active-text="启用"
          inactive-text="停用"
          inline-prompt
          style={switchStyle.value}
          onChange={() => onSwitchChange(scope as any)}
        />
      )
    },
    {
      label: "删除标记",
      prop: "deleted_at",
      width: 90,
      hide: () => !isSuperAdmin.value,
      cellRenderer: ({ row }) => (
        <el-tag size="small" type={row.deleted_at ? "danger" : "info"}>
          {row.deleted_at ? "已删除" : "正常"}
        </el-tag>
      )
    },
    {
      label: "注册时间",
      prop: "created_at",
      minWidth: 160,
      formatter: ({ created_at }) =>
        created_at ? dayjs(created_at).format("YYYY-MM-DD HH:mm:ss") : "-"
    },
    { label: "操作", fixed: "right", width: 200, slot: "operation" }
  ];

  const buttonClass = computed(() => [
    "h-5!",
    "reset-margin",
    "text-gray-500!",
    "dark:text-white!",
    "dark:hover:text-primary!"
  ]);

  const pwdForm = reactive({ newPwd: "", newPwd2: "" });
  /** 密码三重 md5 之后再提交；空着的（不改）原样发空串 */
  const hashed = (pwd: string) => (pwd ? md5(md5(md5(pwd))) : "");
  const pwdProgress = [
    { color: "#e74242", text: "非常弱" },
    { color: "#EFBD47", text: "弱" },
    { color: "#ffa500", text: "一般" },
    { color: "#1bbf1b", text: "强" },
    { color: "#008000", text: "非常强" }
  ];
  const curScore = ref(-1);
  watch(
    pwdForm,
    ({ newPwd }) =>
      (curScore.value = isAllEmpty(newPwd) ? -1 : zxcvbn(newPwd).score)
  );

  function onSwitchChange({ row, index }) {
    ElMessageBox.confirm(
      `确认要<strong>${row.status === 0 ? "停用" : "启用"}</strong>用户 <strong style="color:var(--el-color-primary)">${row.username}</strong> 吗？`,
      "系统提示",
      {
        confirmButtonText: "确定",
        cancelButtonText: "取消",
        type: "warning",
        dangerouslyUseHTMLString: true,
        draggable: true
      }
    )
      .then(async () => {
        switchLoadMap.value[index] = { loading: true };
        try {
          const res = await actAdminSwitch({
            id: row.id,
            status: row.status
          });
          if (res.code !== 0) {
            row.status = row.status === 1 ? 0 : 1;
            message(res.message || "操作失败", { type: "error" });
          }
        } catch {
          row.status = row.status === 1 ? 0 : 1;
        } finally {
          switchLoadMap.value[index] = { loading: false };
        }
      })
      .catch(() => {
        row.status = row.status === 1 ? 0 : 1;
      });
  }

  async function onSearch() {
    loading.value = true;
    try {
      const res = await actAdminList({
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
    } catch {
      message("获取用户列表失败，请检查后端服务", { type: "error" });
    } finally {
      loading.value = false;
    }
  }

  function resetForm(formEl) {
    if (!formEl) return;
    formEl.resetFields();
    treeRef.value?.onTreeReset?.();
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

  function handleSelectionChange(val) {
    selectedNum.value = val.length;
    tableRef.value?.setAdaptive?.();
  }
  function onSelectionCancel() {
    selectedNum.value = 0;
    tableRef.value?.getTableRef()?.clearSelection();
  }
  async function onbatchDel() {
    const curSelected = tableRef.value?.getTableRef()?.getSelectionRows() ?? [];
    if (!curSelected.length) return;
    const ids = getKeyList(curSelected, "id");
    const res = await actAdminDel({ ids }).catch(() => null);
    if (res?.code === 0) {
      message(`已删除 ${ids.length} 条用户数据`, { type: "success" });
      tableRef.value?.getTableRef()?.clearSelection();
      onSearch();
    } else {
      message(res?.message || "删除失败", { type: "error" });
    }
  }
  async function handleDelete(row) {
    const res = await actAdminDel({ ids: [row.id] }).catch(() => null);
    if (res?.code === 0) {
      message("删除成功", { type: "success" });
      onSearch();
    } else {
      message(res?.message || "删除失败", { type: "error" });
    }
  }

  function openDialog(title = "新增", row?: any) {
    addDialog({
      title: `${title}管理员`,
      props: {
        formInline: {
          title,
          id: row?.id,
          username: row?.username ?? "",
          real_name: row?.real_name ?? "",
          password: "",
          gender: row?.gender ?? 0,
          phone: row?.phone ?? "",
          email: row?.email ?? "",
          status: row?.status ?? 1,
          roles: row?.roles ? [...row.roles] : [],
          avatar: row?.avatar ?? "",
          dept_id: row?.dept_id ?? 0,
          deptOptions: treeData.value,
          roleOptions: roleOptions.value
        } as FormItemProps
      },
      width: "50%",
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
          const res =
            title === "新增"
              ? await actAdminAdd({ ...curData }).catch(() => null)
              : await actAdminEdit({ ...curData }).catch(() => null);
          if (res?.code === 0) {
            message(`${title}管理员成功`, { type: "success" });
            done();
            onSearch();
          } else {
            message(res?.message || `${title}失败`, { type: "error" });
          }
        });
      }
    });
  }

  const cropRef = ref();
  function handleUpload(row) {
    avatarInfo.value = null;
    addDialog({
      title: "裁剪、上传头像",
      width: "40%",
      closeOnClickModal: false,
      fullscreen: deviceDetection(),
      contentRenderer: () =>
        h(ReCropperPreview, {
          ref: cropRef,
          imgSrc: row.avatar || "",
          onCropper: (info: any) => (avatarInfo.value = info)
        }),
      beforeSure: async done => {
        if (!avatarInfo.value?.base64) {
          message("请先裁剪头像", { type: "warning" });
          return;
        }
        const res = await actAdminSetAvatar({
          id: row.id,
          avatar: avatarInfo.value.base64
        }).catch(() => null);
        if (res?.code === 0) {
          message("头像保存成功", { type: "success" });
          done();
          onSearch();
        } else {
          message(res?.message || "保存失败", { type: "error" });
        }
      },
      closeCallBack: () => cropRef.value?.hidePopover?.()
    });
  }

  function handleReset(row) {
    pwdForm.newPwd = "";
    pwdForm.newPwd2 = "";
    addDialog({
      title: `重置 ${row.username} 的密码`,
      width: "30%",
      draggable: true,
      closeOnClickModal: false,
      fullscreen: deviceDetection(),
      contentRenderer: () => (
        <>
          <ElForm ref={ruleFormRef} model={pwdForm} label-width="82px">
            <ElFormItem
              label="登录密码"
              prop="newPwd"
              rules={[
                {
                  validator: (_rule, _value, callback) =>
                    pwdForm.newPwd || pwdForm.newPwd2
                      ? callback()
                      : callback(new Error("登录密码、二次密码至少填一个")),
                  trigger: "blur"
                }
              ]}
            >
              <ElInput
                clearable
                show-password
                type="password"
                v-model={pwdForm.newPwd}
                placeholder="新的登录密码，不改就留空"
              />
            </ElFormItem>
            <ElFormItem label="二次密码" prop="newPwd2">
              <ElInput
                clearable
                show-password
                type="password"
                v-model={pwdForm.newPwd2}
                placeholder="新的二次密码，不改就留空"
                onInput={() => ruleFormRef.value?.clearValidate("newPwd")}
              />
              <div class="text-xs/5 text-gray-400">
                一个账号只能有一处登录：已在别处登录时再登录要输二次密码。没有单独设置过的，二次密码就是登录密码。
              </div>
            </ElFormItem>
          </ElForm>
          <div class="my-4 flex">
            {pwdProgress.map(({ color, text }, idx) => (
              <div class="flex-1" style={{ marginLeft: idx !== 0 ? "4px" : 0 }}>
                <ElProgress
                  striped
                  striped-flow
                  duration={curScore.value === idx ? 6 : 0}
                  percentage={curScore.value >= idx ? 100 : 0}
                  color={color}
                  stroke-width={10}
                  show-text={false}
                />
                <p
                  class="text-center text-xs mt-1"
                  style={{ color: curScore.value === idx ? color : "" }}
                >
                  {text}
                </p>
              </div>
            ))}
          </div>
        </>
      ),
      closeCallBack: () => {
        pwdForm.newPwd = "";
        pwdForm.newPwd2 = "";
      },
      beforeSure: done => {
        ruleFormRef.value?.validate(async valid => {
          if (!valid) return;
          const res = await actAdminResetPwd({
            id: row.id,
            new_pwd: hashed(pwdForm.newPwd),
            new_pwd2: hashed(pwdForm.newPwd2)
          }).catch(() => null);
          if (res?.code === 0) {
            message(`已成功重置 ${row.username} 的密码`, { type: "success" });
            done();
          } else {
            message(res?.message || "重置失败", { type: "error" });
          }
        });
      }
    });
  }

  function handleRole(row) {
    addDialog({
      title: `分配 ${row.username} 的角色`,
      props: {
        formInline: {
          id: row.id,
          username: row.username,
          real_name: row.real_name ?? "",
          roles: row.roles ? [...row.roles] : [],
          roleOptions: roleOptions.value
        } as RoleFormItemProps
      },
      width: "420px",
      draggable: true,
      fullscreen: deviceDetection(),
      fullscreenIcon: true,
      closeOnClickModal: false,
      contentRenderer: () => h(roleForm),
      beforeSure: async (done, { options }) => {
        const curData = options.props.formInline as RoleFormItemProps;
        const res = await actAdminRoleSave({
          id: curData.id,
          roles: curData.roles
        }).catch(() => null);
        if (res?.code === 0) {
          message("角色分配成功", { type: "success" });
          done();
          onSearch();
        } else {
          message(res?.message || "保存失败", { type: "error" });
        }
      }
    });
  }

  function onTreeSelect({ id, selected }) {
    form.dept_id = selected ? id : 0;
    onSearch();
  }

  onMounted(async () => {
    treeLoading.value = true;
    try {
      const [rolesRes, deptRes] = await Promise.all([
        actAdminRoleList({ parent_id: 1 }),
        actAdminDeptList()
      ]);
      if (rolesRes.code === 0) roleOptions.value = rolesRes.data ?? [];
      if (deptRes.code === 0) {
        deptOptions.value = deptRes.data ?? [];
        treeData.value = handleTree(deptRes.data ?? [], "id", "parent_id");
      }
    } catch (e) {
      console.error(e);
    } finally {
      treeLoading.value = false;
    }
    onSearch();
  });

  return {
    form,
    loading,
    columns,
    dataList,
    treeData,
    treeLoading,
    selectedNum,
    pagination,
    buttonClass,
    deviceDetection,
    onSearch,
    resetForm,
    onbatchDel,
    openDialog,
    onTreeSelect,
    handleDelete,
    handleUpload,
    handleReset,
    handleRole,
    handleSizeChange,
    onSelectionCancel,
    handleCurrentChange,
    handleSelectionChange
  };
}
