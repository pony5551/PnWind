<script setup lang="ts">
import { ref } from "vue";
import ReCol from "@/components/ReCol";
import type { RoleFormProps } from "../utils/types";

const props = withDefaults(defineProps<RoleFormProps>(), {
  formInline: () => ({
    id: 0,
    username: "",
    real_name: "",
    roles: [],
    roleOptions: []
  })
});

const newFormInline = ref(props.formInline);
</script>

<template>
  <el-form :model="newFormInline" label-width="80px">
    <el-row :gutter="20">
      <re-col :value="24">
        <el-form-item label="用户名">
          <el-input :value="newFormInline.username" disabled />
        </el-form-item>
      </re-col>
      <re-col v-if="newFormInline.real_name" :value="24">
        <el-form-item label="真实姓名">
          <el-input :value="newFormInline.real_name" disabled />
        </el-form-item>
      </re-col>
      <re-col :value="24">
        <el-form-item label="分配角色">
          <el-select
            v-model="newFormInline.roles"
            multiple
            clearable
            collapse-tags
            collapse-tags-tooltip
            placeholder="请选择角色（支持多选）"
            class="w-full"
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
      </re-col>
    </el-row>
  </el-form>
</template>
