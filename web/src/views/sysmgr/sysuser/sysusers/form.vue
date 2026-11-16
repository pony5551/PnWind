<script setup lang="ts">
import { ref, computed } from "vue";
import { formRules, editFormRules } from "./utils/rule";
import type { FormProps } from "./utils/types";
import { USERS_TYPE_MAP, USERS_STATUS_MAP } from "./utils/hook";

const props = withDefaults(defineProps<FormProps>(), {
  formInline: () => ({
    id: 0,
    name: "",
    type: 0,
    status: 0,
    email: "",
    phone: "",
    roles: [],
    roleOptions: [],
    config: "",
    remark: ""
  })
});

const ruleFormRef = ref();
const newFormInline = ref(props.formInline);

// 新增时 id=0，编辑时 id>0
const isEdit = computed(() => newFormInline.value.id > 0);
const activeRules = computed(() => (isEdit.value ? editFormRules : formRules));

function getRef() {
  return ruleFormRef.value;
}

defineExpose({ getRef });
</script>

<template>
  <el-form
    ref="ruleFormRef"
    :model="newFormInline"
    :rules="activeRules"
    label-width="100px"
  >
    <el-form-item label="用户名称" prop="name">
      <el-input
        v-model="newFormInline.name"
        clearable
        placeholder="请输入用户名称"
        :maxlength="128"
        show-word-limit
      />
    </el-form-item>

    <el-form-item label="用户类型" prop="type">
      <el-select
        v-model="newFormInline.type"
        placeholder="请选择用户类型"
        class="w-full!"
      >
        <el-option
          v-for="(label, val) in USERS_TYPE_MAP"
          :key="val"
          :label="label"
          :value="Number(val)"
        />
      </el-select>
    </el-form-item>

    <el-form-item label="用户状态" prop="status">
      <el-select
        v-model="newFormInline.status"
        placeholder="请选择用户状态"
        class="w-full!"
      >
        <el-option
          v-for="(s, val) in USERS_STATUS_MAP"
          :key="val"
          :label="s.label"
          :value="Number(val)"
        />
      </el-select>
    </el-form-item>

    <el-form-item label="用户邮箱" prop="email">
      <el-input
        v-model="newFormInline.email"
        clearable
        placeholder="请输入用户邮箱"
        :maxlength="128"
      />
    </el-form-item>

    <el-form-item label="用户手机" prop="phone">
      <el-input
        v-model="newFormInline.phone"
        clearable
        placeholder="请输入用户手机号"
        :maxlength="20"
      />
    </el-form-item>

    <el-form-item label="用户角色" prop="roles">
      <el-select
        v-model="newFormInline.roles"
        multiple
        clearable
        collapse-tags
        collapse-tags-tooltip
        placeholder="请选择用户角色（支持多选）"
        class="w-full!"
      >
        <el-option
          v-for="item in newFormInline.roleOptions"
          :key="item.code"
          :label="item.name"
          :value="item.code"
        >
          <span>{{ item.name }}</span>
          <span class="ml-2 text-gray-400 text-xs">{{ item.code }}</span>
        </el-option>
      </el-select>
    </el-form-item>

    <el-form-item label="用户配置" prop="config">
      <el-input
        v-model="newFormInline.config"
        type="textarea"
        :rows="3"
        placeholder='JSON格式，例: {"key": "value"}'
      />
    </el-form-item>

    <el-form-item label="备注" prop="remark">
      <el-input
        v-model="newFormInline.remark"
        type="textarea"
        :rows="2"
        clearable
        placeholder="请输入备注（选填）"
      />
    </el-form-item>
  </el-form>
</template>
