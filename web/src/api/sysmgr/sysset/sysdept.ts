import { http } from "@/utils/http";
import { sysdept_url } from "@/api/sysmgr/sysmgr_define";

const pageUrl = sysdept_url;

type Result = {
  code: number;
  message: string;
  data?: Array<any>;
};

/** 部门列表（全量，前端处理树结构） */
export const actDeptList = (data?: object) => {
  return http.request<Result>("post", pageUrl.concat("/list"), { data });
};

/** 新增部门 */
export const actDeptAdd = (data?: object) => {
  return http.request<Result>("post", pageUrl.concat("/add"), { data });
};

/** 修改部门 */
export const actDeptEdit = (data?: object) => {
  return http.request<Result>("post", pageUrl.concat("/edit"), { data });
};

/** 删除部门 */
export const actDeptDel = (data?: object) => {
  return http.request<Result>("post", pageUrl.concat("/del"), { data });
};
