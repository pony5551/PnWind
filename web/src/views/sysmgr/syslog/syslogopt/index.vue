<script setup lang="ts">
import { ref } from "vue";
import { useLogOpt } from "./utils/hook";
import { PureTableBar } from "@/components/RePureTableBar";
import { useRenderIcon } from "@/components/ReIcon/src/hooks";
import Delete from "~icons/ep/delete";
import Refresh from "~icons/ep/refresh";

defineOptions({
  name: "m_syslogopt"
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
  isSuperAdmin,
  userTypeOptions,
  optTypeOptions,
  dateShortcuts,
  onSearch,
  resetForm,
  handleSizeChange,
  handleCurrentChange,
  handleSelectionChange,
  onSelectionCancel,
  onbatchDel,
  onClear
} = useLogOpt(tableRef);
</script>

<template>
  <div class="main">
    <el-form
      ref="formRef"
      :inline="true"
      :model="form"
      class="search-form bg-bg_color w-full pl-8 pt-3 overflow-auto"
    >
      <el-form-item label="用户：" prop="user_id">
        <el-select
          v-model="form.user_id"
          filterable
          allow-create
          default-first-option
          clearable
          placeholder="请选择或输入"
          class="w-44!"
        />
      </el-form-item>
      <el-form-item label="用户类型：" prop="user_type">
        <el-select
          v-model="form.user_type"
          clearable
          placeholder="全部"
          class="w-32!"
        >
          <el-option
            v-for="o in userTypeOptions"
            :key="o.value"
            :label="o.label"
            :value="o.value"
          />
        </el-select>
      </el-form-item>
      <el-form-item label="操作类型：" prop="type">
        <el-select
          v-model="form.type"
          filterable
          allow-create
          default-first-option
          clearable
          placeholder="请选择或输入"
          class="w-40!"
        >
          <el-option
            v-for="o in optTypeOptions"
            :key="o.value"
            :label="o.label"
            :value="o.value"
          />
        </el-select>
      </el-form-item>
      <el-form-item label="关键词：" prop="skey">
        <el-input
          v-model="form.skey"
          placeholder="关键词"
          clearable
          class="w-44!"
        />
      </el-form-item>
      <el-form-item label="时间：" prop="timeRange">
        <el-date-picker
          v-model="form.timeRange"
          type="datetimerange"
          range-separator="→"
          start-placeholder="开始时间"
          end-placeholder="结束时间"
          format="YYYY-MM-DD HH:mm:ss"
          value-format="YYYY-MM-DD HH:mm:ss"
          :default-time="[
            new Date(2000, 0, 1, 0, 0, 0),
            new Date(2000, 0, 1, 23, 59, 59)
          ]"
          :shortcuts="dateShortcuts"
          clearable
          class="w-100!"
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
      title="操作日志"
      :columns="columns"
      :tableRef="tableRef?.getTableRef()"
      @refresh="onSearch"
    >
      <template #buttons>
        <el-button
          v-if="isSuperAdmin"
          type="danger"
          plain
          :icon="useRenderIcon(Delete)"
          @click="onClear"
        >
          清空日志
        </el-button>
      </template>
      <template v-slot="{ size, dynamicColumns }">
        <!-- 多选操作栏：仅超级管理员可见 -->
        <div
          v-if="isSuperAdmin && selectedNum > 0"
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

          <el-popconfirm title="是否确认批量删除?" @confirm="onbatchDel">
            <template #reference>
              <el-button type="danger" size="small" plain>批量删除</el-button>
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
        />
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
