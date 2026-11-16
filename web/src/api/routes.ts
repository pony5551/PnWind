import { http } from "@/utils/http";
import { sysmenus_url } from "@/api/sysmgr/sysmgr_define";
const pageUrl = sysmenus_url;

type Result = {
  code: number;
  message: string;
  data: Array<any>;
};

export const getAsyncRoutes = () => {
  // return http.request<Result>("get", "/get-async-routes");
  const lurl = pageUrl.concat("/getmainmenu");
  return http.request<Result>("get", lurl);
};
