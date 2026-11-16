<script setup lang="ts">
import { ref } from "vue";
import { useTokenList } from "./utils/hook";
import { PureTableBar } from "@/components/RePureTableBar";
import { useRenderIcon } from "@/components/ReIcon/src/hooks";

import Delete from "~icons/ep/delete";
import Refresh from "~icons/ep/refresh";

defineOptions({
  name: "m_systokenlist"
});

const formRef = ref();
const tableRef = ref();

const {
  form,
  loading,
  columns,
  dataList,
  pagination,
  selectedNum,
  onSearch,
  resetForm,
  handleDelete,
  handleSizeChange,
  handleCurrentChange,
  handleSelectionChange,
  onSelectionCancel,
  onBatchDel
} = useTokenList(tableRef);
</script>

<template>
  <div class="main">
    <el-form
      ref="formRef"
      :inline="true"
      :model="form"
      class="search-form bg-bg_color w-full pl-8 pt-3 overflow-auto"
    >
      <el-form-item label="用户ID：" prop="user_id">
        <el-input
          v-model="form.user_id"
          placeholder="请输入用户ID"
          clearable
          class="w-45!"
        />
      </el-form-item>
      <el-form-item label="用户类型：" prop="user_type">
        <el-select
          v-model="form.user_type"
          placeholder="全部"
          clearable
          class="w-36!"
        >
          <el-option :value="1" label="管理员" />
          <el-option :value="2" label="用户" />
        </el-select>
      </el-form-item>
      <el-form-item label="IP地址：" prop="ip">
        <el-input
          v-model="form.ip"
          placeholder="请输入IP地址"
          clearable
          class="w-45!"
        />
      </el-form-item>
      <el-form-item label="登录开始：" prop="starttime">
        <el-date-picker
          v-model="form.starttime"
          type="datetime"
          placeholder="选择开始时间"
          format="YYYY-MM-DD HH:mm:ss"
          value-format="YYYY-MM-DD HH:mm:ss"
          clearable
          class="w-52!"
        />
      </el-form-item>
      <el-form-item label="登录结束：" prop="endtime">
        <el-date-picker
          v-model="form.endtime"
          type="datetime"
          placeholder="选择结束时间"
          format="YYYY-MM-DD HH:mm:ss"
          value-format="YYYY-MM-DD HH:mm:ss"
          clearable
          class="w-52!"
        />
      </el-form-item>
      <el-form-item>
        <el-button
          type="primary"
          :icon="useRenderIcon('ri/search-line')"
          :loading="loading"
          @click="onSearch"
        >
          搜索
        </el-button>
        <el-button :icon="useRenderIcon(Refresh)" @click="resetForm(formRef)">
          重置
        </el-button>
      </el-form-item>
    </el-form>

    <PureTableBar
      title="在线用户"
      :columns="columns"
      :tableRef="tableRef?.getTableRef()"
      @refresh="onSearch"
    >
      <template v-slot="{ size, dynamicColumns }">
        <!-- 多选操作栏 -->
        <div
          v-if="selectedNum > 0"
          v-motion-fade
          class="w-full mb-2 px-4 py-2 flex items-center gap-x-3 gap-y-2 flex-wrap bg-(--el-fill-color-light) border-b-2 border-(--el-color-primary-light-7)"
        >
          <div class="flex items-center gap-1.5 shrink-0">
            <el-tag type="primary" size="small" effect="dark" round>
              已选 {{ selectedNum }} 项
            </el-tag>
            <el-button
              type="primary"
              link
              size="small"
              @click="onSelectionCancel"
            >
              取消选择
            </el-button>
          </div>
          <el-divider direction="vertical" class="h-5! mx-0" />
          <el-popconfirm title="是否确认批量强制下线？" @confirm="onBatchDel">
            <template #reference>
              <el-button type="danger" size="small" plain
                >批量强制下线</el-button
              >
            </template>
          </el-popconfirm>
        </div>

        <pure-table
          ref="tableRef"
          row-key="id"
          border
          align-whole="center"
          showOverflowTooltip
          table-layout="auto"
          :loading="loading"
          :size="size"
          adaptive
          :adaptiveConfig="{ offsetBottom: 108 }"
          :data="dataList"
          :columns="dynamicColumns"
          :pagination="{ ...pagination, size }"
          :header-cell-style="{
            background: 'var(--el-fill-color-light)',
            color: 'var(--el-text-color-primary)'
          }"
          @selection-change="handleSelectionChange"
          @page-size-change="handleSizeChange"
          @page-current-change="handleCurrentChange"
        >
          <template #operation="{ row }">
            <el-popconfirm
              :title="`确认强制下线用户 ${row.user_id}？`"
              @confirm="handleDelete(row)"
            >
              <template #reference>
                <el-button
                  class="reset-margin"
                  link
                  type="danger"
                  :size="size"
                  :icon="useRenderIcon(Delete)"
                >
                  下线
                </el-button>
              </template>
            </el-popconfirm>
          </template>
        </pure-table>
      </template>
    </PureTableBar>
  </div>
</template>

<style lang="scss" scoped>
.main-content {
  margin: 24px 24px 0 !important;
}

:deep(.el-table__inner-wrapper::before) {
  height: 0;
}

.search-form {
  :deep(.el-form-item) {
    margin-bottom: 12px;
  }
}
</style>
