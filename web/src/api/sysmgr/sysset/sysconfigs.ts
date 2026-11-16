import { http } from "@/utils/http";
import { sysconfigs_url } from "@/api/sysmgr/sysmgr_define";

const pageUrl = sysconfigs_url;

type Result = {
  code: number;
  message: string;
  data?: any;
};

/** 基础设置列表 */
export const actConfigList = (data?: object) =>
  http.request<Result>("post", pageUrl + "/list", { data });

/** 新增配置项 */
export const actConfigAdd = (data?: object) =>
  http.request<Result>("post", pageUrl + "/add", { data });

/** 修改配置值 */
export const actConfigEdit = (data?: object) =>
  http.request<Result>("post", pageUrl + "/edit", { data });

/** 删除配置项 */
export const actConfigDel = (data?: object) =>
  http.request<Result>("post", pageUrl + "/del", { data });

/** 重新加载系统配置到内存 */
export const actLoadSysConfig = (data?: object) =>
  http.request<Result>("post", pageUrl + "/loadsysconfig", { data });

/** 获取系统配置（前端初始化使用） */
export const actGetSysConfig = () =>
  http.request<Result>("get", pageUrl + "/getsysconfig");
