<script setup lang="ts">
import { ref } from "vue";
import { useUsers, USERS_TYPE_MAP, USERS_STATUS_MAP } from "./utils/hook";
import { PureTableBar } from "@/components/RePureTableBar";
import { useRenderIcon } from "@/components/ReIcon/src/hooks";

import EditPen from "~icons/ep/edit-pen";
import Refresh from "~icons/ep/refresh";
import AddFill from "~icons/ri/add-circle-line";
import Password from "~icons/ep/key";
import Delete from "~icons/ep/delete";

defineOptions({
  name: "m_sysusers"
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
  onSearch,
  resetForm,
  openDialog,
  openResetPwdDialog,
  handleSizeChange,
  handleCurrentChange,
  handleSelectionChange,
  onSelectionCancel,
  onbatchDel
} = useUsers(tableRef);
</script>

<template>
  <div class="main">
    <el-form
      ref="formRef"
      :inline="true"
      :model="form"
      class="search-form bg-bg_color w-full pl-8 pt-3 overflow-auto"
    >
      <el-form-item label="用户编号：" prop="user_id">
        <el-input
          v-model="form.user_id"
          placeholder="请输入用户编号"
          clearable
          class="w-45!"
        />
      </el-form-item>
      <el-form-item label="用户名称：" prop="name">
        <el-input
          v-model="form.name"
          placeholder="请输入用户名称"
          clearable
          class="w-45!"
        />
      </el-form-item>
      <el-form-item label="用户邮箱：" prop="email">
        <el-input
          v-model="form.email"
          placeholder="请输入邮箱"
          clearable
          class="w-45!"
        />
      </el-form-item>
      <el-form-item label="用户手机：" prop="phone">
        <el-input
          v-model="form.phone"
          placeholder="请输入手机号"
          clearable
          class="w-45!"
        />
      </el-form-item>
      <el-form-item label="用户类型：" prop="type">
        <el-select
          v-model="form.type"
          placeholder="全部"
          clearable
          class="w-45!"
        >
          <el-option :value="0" label="全部" />
          <el-option
            v-for="(label, val) in USERS_TYPE_MAP"
            :key="val"
            :label="label"
            :value="Number(val)"
          />
        </el-select>
      </el-form-item>
      <el-form-item label="用户状态：" prop="status">
        <el-select
          v-model="form.status"
          placeholder="全部"
          clearable
          class="w-45!"
        >
          <el-option
            v-for="(s, val) in USERS_STATUS_MAP"
            :key="val"
            :label="s.label"
            :value="Number(val)"
          />
        </el-select>
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

    <PureTableBar title="用户管理" :columns="columns" @refresh="onSearch">
      <template #buttons>
        <el-button
          type="primary"
          :icon="useRenderIcon(AddFill)"
          @click="openDialog()"
        >
          新增用户
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
        >
          <template #operation="{ row }">
            <el-button
              class="reset-margin"
              link
              type="primary"
              :size="size"
              :icon="useRenderIcon(EditPen)"
              @click="openDialog('修改', row)"
            >
              修改
            </el-button>
            <el-button
              class="reset-margin"
              link
              type="warning"
              :size="size"
              :icon="useRenderIcon(Password)"
              @click="openResetPwdDialog(row)"
            >
              重置密码
            </el-button>
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

.search-form {
  :deep(.el-form-item) {
    margin-bottom: 12px;
  }
}
</style>
