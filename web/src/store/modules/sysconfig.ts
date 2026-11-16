import { defineStore } from "pinia";
import { store } from "../utils";

interface SysconfigState {
  /** 默认每页显示条数（来自 web.pageSize） */
  pageSize: number;
  /** 每页条数选项（来自 web.pageSizeOptions，逗号分隔字符串解析） */
  pageSizeOptions: number[];
  /** 原始配置 key-value */
  rawConfig: Record<string, string>;
}

export const useSysconfigStore = defineStore("pure-sysconfig", {
  state: (): SysconfigState => ({
    pageSize: 20,
    pageSizeOptions: [20, 50, 100, 200, 500, 1000],
    rawConfig: {}
  }),
  getters: {
    /** 获取任意配置项 */
    getConfig:
      state =>
      (key: string): string => {
        return state.rawConfig[key] ?? "";
      }
  },
  actions: {
    /** 解析并存储系统配置 */
    setSysConfig(data: Record<string, string>) {
      this.rawConfig = data ?? {};

      // web.pageSize → 默认分页大小
      const ps = parseInt(data?.["web.pageSize"]);
      if (!isNaN(ps) && ps > 0) {
        this.pageSize = ps;
      }

      // web.pageSizeOptions → 分页大小可选项（逗号分隔）
      const pso = data?.["web.pageSizeOptions"];
      if (pso) {
        const options = pso
          .split(",")
          .map(s => parseInt(s.trim()))
          .filter(n => !isNaN(n) && n > 0);
        if (options.length > 0) {
          this.pageSizeOptions = options;
        }
      }
    }
  }
});

export function useSysconfigStoreHook() {
  return useSysconfigStore(store);
}
