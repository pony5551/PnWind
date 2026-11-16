<script setup lang="ts">
import { ref, reactive, onMounted } from "vue";
import { PureTableBar } from "@/components/RePureTableBar";
import { useRenderIcon } from "@/components/ReIcon/src/hooks";
import { message } from "@/utils/message";
import Delete from "~icons/ep/delete";
import EditPen from "~icons/ep/edit-pen";
import Refresh from "~icons/ep/refresh";
import AddFill from "~icons/ri/add-circle-line";
import {
  actConfigList,
  actConfigAdd,
  actConfigEdit,
  actConfigDel,
  actLoadSysConfig
} from "@/api/sysmgr/sysset/sysconfigs";

defineOptions({ name: "m_sysconfigs" });

// ─── 状态 ────────────────────────────────────────────────────────────────────

const loading = ref(false);
const dataList = ref<any[]>([]);
const editingId = ref<number | null>(null);
const editingValue = ref("");
const tableRef = ref();

const searchForm = reactive({ sort: "" });

// 新增对话框状态
const addVisible = ref(false);
const addSaving = ref(false);
const addForm = reactive({
  sort: "",
  key: "",
  value: "",
  remark: ""
});

// ─── 表格列定义 ───────────────────────────────────────────────────────────────

const columns: TableColumnList = [
  { type: "selection", width: 40, reserveSelection: true, align: "center" },
  {
    label: "变量标题",
    prop: "key",
    minWidth: 130,
    align: "left",
    showOverflowTooltip: true
  },
  {
    label: "变量值",
    prop: "value",
    minWidth: 160,
    align: "left",
    slot: "value"
  },
  {
    label: "备注",
    prop: "remark",
    minWidth: 120,
    align: "left",
    showOverflowTooltip: true
  },
  {
    label: "变量名",
    prop: "varname",
    width: 180,
    align: "left",
    slot: "varname"
  },
  {
    label: "操作",
    fixed: "right",
    width: 120,
    align: "center",
    slot: "operation"
  }
];

// ─── 数据加载 ─────────────────────────────────────────────────────────────────

async function onSearch() {
  loading.value = true;
  try {
    const res = await actConfigList({
      sort: searchForm.sort,
      pageSize: 5000,
      currentPage: 1
    });
    if (res.code === 0) {
      dataList.value = res.data?.list ?? [];
    } else {
      message(res.message || "加载失败", { type: "error" });
    }
  } finally {
    loading.value = false;
  }
}

function resetForm() {
  searchForm.sort = "";
  onSearch();
}

// ─── 行样式 ───────────────────────────────────────────────────────────────────

function rowClassName({ row }: { row: any }) {
  return row.sort === "sys" ? "sys-row" : "";
}

// ─── 内联编辑 ─────────────────────────────────────────────────────────────────

function startEdit(row: any) {
  editingId.value = row.config_id;
  editingValue.value = row.value;
}

async function confirmEdit(row: any) {
  if (!editingValue.value && editingValue.value !== "0") {
    message("修改值不能为空", { type: "warning" });
    return;
  }
  try {
    const res = await actConfigEdit({
      config_id: row.config_id,
      value: editingValue.value
    });
    if (res.code === 0) {
      row.value = editingValue.value;
      message("修改成功", { type: "success" });
    } else {
      message(res.message || "修改失败", { type: "error" });
    }
  } finally {
    editingId.value = null;
  }
}

function cancelEdit() {
  editingId.value = null;
}

// ─── 删除 ─────────────────────────────────────────────────────────────────────

async function handleDelete(row: any) {
  const res = await actConfigDel({ ids: [row.config_id] });
  if (res.code === 0) {
    message("删除成功", { type: "success" });
    onSearch();
  } else {
    message(res.message || "删除失败", { type: "error" });
  }
}

// ─── 新增对话框 ───────────────────────────────────────────────────────────────

function openAddDialog() {
  addForm.sort = "";
  addForm.key = "";
  addForm.value = "";
  addForm.remark = "";
  addVisible.value = true;
}

async function handleAddSave() {
  if (!addForm.sort) {
    message("分组不能为空", { type: "warning" });
    return;
  }
  if (!addForm.key) {
    message("键不能为空", { type: "warning" });
    return;
  }
  if (!addForm.value) {
    message("值不能为空", { type: "warning" });
    return;
  }
  addSaving.value = true;
  try {
    const res = await actConfigAdd({ ...addForm });
    if (res.code === 0) {
      message("新增成功", { type: "success" });
      addVisible.value = false;
      onSearch();
    } else {
      message(res.message || "新增失败", { type: "error" });
    }
  } finally {
    addSaving.value = false;
  }
}

// ─── 重新加载系统配置 ─────────────────────────────────────────────────────────

async function handleReload() {
  const res = await actLoadSysConfig({});
  if (res.code === 0) {
    message("系统配置已重新加载", { type: "success" });
  } else {
    message(res.message || "加载失败", { type: "error" });
  }
}

onMounted(onSearch);
</script>

<template>
  <div class="main">
    <!-- 搜索栏 -->
    <el-form
      :inline="true"
      :model="searchForm"
      class="search-form bg-bg_color w-full pl-8 pt-3 overflow-auto"
    >
      <el-form-item label="分组：" prop="sort">
        <el-input
          v-model="searchForm.sort"
          placeholder="按分组筛选（如 sys / web）"
          clearable
          class="w-50!"
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
        <el-button :icon="useRenderIcon(Refresh)" @click="resetForm">
          重置
        </el-button>
      </el-form-item>
    </el-form>

    <!-- 表格区域 -->
    <PureTableBar title="基础设置" :columns="columns" @refresh="onSearch">
      <template #buttons>
        <el-button :icon="useRenderIcon(Refresh)" @click="handleReload">
          重新加载
        </el-button>
        <el-button
          type="primary"
          :icon="useRenderIcon(AddFill)"
          @click="openAddDialog"
        >
          新增
        </el-button>
      </template>

      <template v-slot="{ size, dynamicColumns }">
        <pure-table
          ref="tableRef"
          row-key="config_id"
          border
          showOverflowTooltip
          table-layout="auto"
          :loading="loading"
          :size="size"
          adaptive
          :adaptiveConfig="{ offsetBottom: 56 }"
          :data="dataList"
          :columns="dynamicColumns"
          :row-class-name="rowClassName"
          :header-cell-style="{
            background: 'var(--el-fill-color-light)',
            color: 'var(--el-text-color-primary)'
          }"
        >
          <!-- 变量值：内联编辑 -->
          <template #value="{ row }">
            <template v-if="editingId === row.config_id">
              <div class="flex items-center gap-1">
                <el-input
                  v-model="editingValue"
                  size="small"
                  class="flex-1"
                  @keyup.enter="confirmEdit(row)"
                  @keyup.esc="cancelEdit"
                />
                <el-button
                  type="primary"
                  size="small"
                  link
                  @click="confirmEdit(row)"
                >
                  确认
                </el-button>
                <el-button size="small" link @click="cancelEdit">
                  取消
                </el-button>
              </div>
            </template>
            <template v-else>
              <span class="mr-1">{{ row.value || "-" }}</span>
              <el-icon
                v-if="row.sort !== 'sys'"
                class="cursor-pointer text-primary hover:opacity-70"
                @click="startEdit(row)"
              >
                <component :is="useRenderIcon(EditPen)" />
              </el-icon>
            </template>
          </template>

          <!-- 变量名：{sort.key} 格式 -->
          <template #varname="{ row }">
            <el-tag type="info" size="small" effect="plain">
              {{ "{" + row.sort + "." + row.key + "}" }}
            </el-tag>
          </template>

          <!-- 操作列 -->
          <template #operation="{ row }">
            <template v-if="row.sort !== 'sys'">
              <el-popconfirm
                :title="'确认删除 {' + row.sort + '.' + row.key + '} 吗？'"
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
                    删除
                  </el-button>
                </template>
              </el-popconfirm>
            </template>
            <template v-else>
              <span class="text-(--el-text-color-placeholder) text-xs">
                系统项
              </span>
            </template>
          </template>
        </pure-table>
      </template>
    </PureTableBar>

    <!-- 新增对话框 -->
    <el-dialog
      v-model="addVisible"
      title="新增配置项"
      width="480px"
      draggable
      destroy-on-close
    >
      <el-form :model="addForm" label-width="82px">
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="分组">
              <el-input
                v-model="addForm.sort"
                placeholder="如 web / app"
                clearable
              />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="键">
              <el-input
                v-model="addForm.key"
                placeholder="如 siteName"
                clearable
              />
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="值">
              <el-input
                v-model="addForm.value"
                placeholder="配置值"
                clearable
              />
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="备注">
              <el-input
                v-model="addForm.remark"
                placeholder="备注说明（可选）"
                clearable
              />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
      <template #footer>
        <el-button @click="addVisible = false">取消</el-button>
        <el-button type="primary" :loading="addSaving" @click="handleAddSave">
          确定
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style lang="scss" scoped>
.search-form {
  :deep(.el-form-item) {
    margin-bottom: 12px;
  }
}

.main-content {
  margin: 24px 24px 0 !important;
}

/* sys 分组行：浅绿背景 */
:deep(.sys-row) {
  --el-table-tr-bg-color: #f0f9eb;

  &:hover > td {
    background-color: #e8f5e1 !important;
  }
}
</style>
