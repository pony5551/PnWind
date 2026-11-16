<script setup lang="ts">
import { ref, computed } from "vue";
import { formRules } from "./utils/rule";
import { FormProps } from "./utils/types";

const props = withDefaults(defineProps<FormProps>(), {
  formInline: () => ({
    higherRoleOptions: [],
    id: 0,
    parent_id: 0,
    name: "",
    code: "",
    sort: 99,
    status: 1,
    remark: ""
  })
});

const ruleFormRef = ref();
const newFormInline = ref(props.formInline);

/** 修改顶级角色时，上级角色和角色标识不可编辑 */
const isTopLevelEdit = computed(
  () => newFormInline.value.id !== 0 && newFormInline.value.parent_id === 0
);

function getRef() {
  return ruleFormRef.value;
}

defineExpose({ getRef });
</script>

<template>
  <el-form
    ref="ruleFormRef"
    :model="newFormInline"
    :rules="formRules"
    label-width="82px"
  >
    <re-col v-show="false">
      <el-form-item label="id" prop="id">
        <el-input v-model="newFormInline.id" />
      </el-form-item>
    </re-col>

    <el-form-item label="上级角色">
      <el-cascader
        v-model="newFormInline.parent_id"
        class="w-full"
        :disabled="isTopLevelEdit"
        :options="newFormInline.higherRoleOptions"
        :props="{
          value: 'id',
          label: 'name',
          emitPath: false,
          checkStrictly: true
        }"
        clearable
        filterable
        placeholder="请选择上级角色（可不选）"
        @clear="newFormInline.parent_id = 0"
      >
        <template #default="{ node, data }">
          <span>{{ data.name }}</span>
          <span v-if="!node.isLeaf"> ({{ data.children.length }}) </span>
        </template>
      </el-cascader>
    </el-form-item>

    <el-form-item label="角色名称" prop="name">
      <el-input
        v-model="newFormInline.name"
        clearable
        placeholder="请输入角色名称"
      />
    </el-form-item>

    <el-form-item label="角色标识" prop="code">
      <el-input
        v-model="newFormInline.code"
        :disabled="isTopLevelEdit"
        clearable
        placeholder="请输入角色标识"
      />
    </el-form-item>

    <el-form-item label="排序" prop="sort">
      <el-input-number
        v-model="newFormInline.sort"
        class="w-full!"
        :min="1"
        :max="9999"
        controls-position="right"
      />
    </el-form-item>

    <el-form-item label="状态" prop="status">
      <el-switch
        v-model="newFormInline.status"
        :active-value="1"
        :inactive-value="0"
        active-text="已启用"
        inactive-text="已停用"
        inline-prompt
      />
    </el-form-item>

    <el-form-item label="备注">
      <el-input
        v-model="newFormInline.remark"
        placeholder="请输入备注信息"
        type="textarea"
      />
    </el-form-item>
  </el-form>
</template>
