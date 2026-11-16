<script setup lang="ts">
import { ref } from "vue";
import { resetPwdRules } from "./utils/rule";
import type { ResetPwdFormDialogProps } from "./utils/types";

const props = withDefaults(defineProps<ResetPwdFormDialogProps>(), {
  formInline: () => ({
    id: 0,
    password: "",
    pay_password: ""
  })
});

const ruleFormRef = ref();
const newFormInline = ref(props.formInline);

function getRef() {
  return ruleFormRef.value;
}

defineExpose({ getRef });
</script>

<template>
  <el-form
    ref="ruleFormRef"
    :model="newFormInline"
    :rules="resetPwdRules"
    label-width="100px"
  >
    <el-alert
      title="仅填写需要重置的密码，留空则不修改"
      type="info"
      :closable="false"
      class="mb-4"
    />

    <el-form-item label="登录密码" prop="password">
      <el-input
        v-model="newFormInline.password"
        type="password"
        show-password
        clearable
        placeholder="留空则不修改登录密码"
      />
    </el-form-item>

    <el-form-item label="资金密码" prop="pay_password">
      <el-input
        v-model="newFormInline.pay_password"
        type="password"
        show-password
        clearable
        placeholder="留空则不修改资金密码"
      />
    </el-form-item>
  </el-form>
</template>
