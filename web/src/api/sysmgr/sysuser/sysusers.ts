import { http } from "@/utils/http";
import type { Result, ResultTable } from "@/api/types";

const pageUrl = "/web/sysmgr/sysuser/sysusers";

export const actUsersList = (data?) =>
  http.request<ResultTable>("post", pageUrl.concat("/list"), { data });

export const actUsersAdd = (data?) =>
  http.request<Result>("post", pageUrl.concat("/add"), { data });

export const actUsersEdit = (data?) =>
  http.request<Result>("post", pageUrl.concat("/edit"), { data });

export const actUsersResetpwd = (data?) =>
  http.request<Result>("post", pageUrl.concat("/resetpwd"), { data });

export const actUsersDel = (data?) =>
  http.request<Result>("post", pageUrl.concat("/del"), { data });

export const actUsersRoleList = (data?) =>
  http.request<Result>("post", pageUrl.concat("/rolelist"), { data });
