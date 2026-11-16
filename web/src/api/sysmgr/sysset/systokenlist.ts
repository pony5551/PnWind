import { http } from "@/utils/http";
import { systokenlist_url } from "@/api/sysmgr/sysmgr_define";

const pageUrl = systokenlist_url;

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

/** 在线用户列表（分页） */
export const actTokenList = (data?: object) =>
  http.request<ResultTable>("post", pageUrl.concat("/list"), { data });

/** 强制下线（删除Token记录） */
export const actTokenDel = (data?: object) =>
  http.request<Result>("post", pageUrl.concat("/del"), { data });
