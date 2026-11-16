import { http } from "@/utils/http";
import { sysadmin_url } from "@/api/sysmgr/sysmgr_define";

const pageUrl = sysadmin_url;

type Result = {
  code: number;
  message: string;
  data?: any;
};

type ResultTable = {
  code: number;
  message: string;
  data?: {
    list: Array<any>;
    total?: number;
    pageSize?: number;
    currentPage?: number;
  };
};

/** 用户列表 */
export const actAdminList = (data?: object) =>
  http.request<ResultTable>("post", pageUrl.concat("/list"), { data });

/** 新增用户 */
export const actAdminAdd = (data?: object) =>
  http.request<Result>("post", pageUrl.concat("/add"), { data });

/** 修改用户 */
export const actAdminEdit = (data?: object) =>
  http.request<Result>("post", pageUrl.concat("/edit"), { data });

/** 删除用户 */
export const actAdminDel = (data?: object) =>
  http.request<Result>("post", pageUrl.concat("/del"), { data });

/** 切换用户状态 */
export const actAdminSwitch = (data?: object) =>
  http.request<Result>("post", pageUrl.concat("/switch"), { data });

/** 重置密码 */
export const actAdminResetPwd = (data?: object) =>
  http.request<Result>("post", pageUrl.concat("/resetpwd"), { data });

/** 保存 base64 头像 */
export const actAdminSetAvatar = (data?: object) =>
  http.request<Result>("post", pageUrl.concat("/setavatar"), { data });

/** 获取角色列表（用于下拉选择，可按上级角色 parent_id 过滤）*/
export const actAdminRoleList = (data?: object) =>
  http.request<Result>("post", pageUrl.concat("/rolelist"), { data });

/** 保存用户角色分配 */
export const actAdminRoleSave = (data?: object) =>
  http.request<Result>("post", pageUrl.concat("/rolesave"), { data });

/** 获取部门列表（用于左侧树及表单下拉）*/
export const actAdminDeptList = () =>
  http.request<Result>("post", pageUrl.concat("/deptlist"), {});
