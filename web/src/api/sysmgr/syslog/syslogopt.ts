import { http } from "@/utils/http";
import { syslogopt_url } from "@/api/sysmgr/sysmgr_define";

const pageUrl = syslogopt_url;

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

type Result = {
  code: number;
  message: string;
  data?: any;
};

/** 操作日志列表（分页） */
export const actLogOptList = (data?: object) =>
  http.request<ResultTable>("post", pageUrl.concat("/list"), { data });

/** 批量删除操作日志（仅限超级管理员） */
export const actLogOptDel = (data?: object) =>
  http.request<Result>("post", pageUrl.concat("/del"), { data });

/** 清空操作日志（仅限超级管理员） */
export const actLogOptClear = () =>
  http.request<Result>("post", pageUrl.concat("/clear"), {});
