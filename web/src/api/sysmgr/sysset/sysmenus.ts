import { http } from "@/utils/http";
import { sysmenus_url } from "@/api/sysmgr/sysmgr_define";
const pageUrl = sysmenus_url;

type Result = {
  code: number;
  message: string;
  data?: Array<any>;
};

/** 获取系统管理-菜单管理列表 */
export const getMenuList = (data?: object) => {
  return http.request<Result>("post", "/menu", { data });
};

/** 获取角色菜单 */
export const actRoleMenuList = (data?: object) => {
  const lurl = pageUrl.concat("/menulist");
  return http.request<Result>("post", lurl, { data });
};

/** 系统菜单列表 */
export const actMenuList = (data?: object) => {
  const lurl = pageUrl.concat("/list");
  return http.request<Result>("post", lurl, {
    data
  });
};

/** 新增系统菜单 */
export const actMenuAdd = (data?: object) => {
  const lurl = pageUrl.concat("/add");
  return http.request<Result>("post", lurl, {
    data
  });
};

/** 编辑系统菜单 */
export const actMenuEdit = (data?: object) => {
  const lurl = pageUrl.concat("/edit");
  return http.request<Result>("post", lurl, {
    data
  });
};

/** 删除系统菜单 */
export const actMenuDel = (data?: object) => {
  const lurl = pageUrl.concat("/del");
  return http.request<Result>("post", lurl, {
    data
  });
};
