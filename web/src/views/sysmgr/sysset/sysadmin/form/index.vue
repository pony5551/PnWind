<script setup lang="ts">
import { h, ref, computed } from "vue";
import ReCol from "@/components/ReCol";
import { message } from "@/utils/message";
import { addDialog } from "@/components/ReDialog";
import { deviceDetection } from "@pureadmin/utils";
import ReCropperPreview from "@/components/ReCropperPreview";
import { formRules } from "../utils/rule";
import type { FormProps } from "../utils/types";
import { usePublicHooks } from "../../hooks";

const props = withDefaults(defineProps<FormProps>(), {
  formInline: () => ({
    title: "新增",
    username: "",
    real_name: "",
    password: "",
    gender: -1,
    phone: "",
    email: "",
    status: 1,
    roles: [],
    avatar: "",
    dept_id: 0,
    deptOptions: [],
    roleOptions: []
  })
});

const ruleFormRef = ref();
const { switchStyle } = usePublicHooks();
const newFormInline = ref(props.formInline);

const isAdd = computed(() => newFormInline.value.title === "新增");

// 头像预览
const avatarPreview = computed(() => newFormInline.value.avatar || "");

/** 头像裁剪后缩放成 192×192（表单里最大显示 64px，按 3 倍屏）；这个尺寸的 PNG 最大也不超过后端 204800 个字符的限制 */
const avatarCanvas = {
  width: 192,
  height: 192,
  imageSmoothingQuality: "high"
} as const;

/** 选好图片后打开裁剪框；浏览器打不开的文件（非图片、HEIC、TIFF、损坏的图片）直接提示 */
function onAvatarChange(e: Event) {
  const input = e.target as HTMLInputElement;
  const file = input.files?.[0];
  input.value = ""; // 清空，取消裁剪后还能再选同一张图片
  if (!file) return;
  const reader = new FileReader();
  reader.onload = ev => {
    const src = ev.target?.result as string;
    const img = new Image();
    img.onload = () => cropAvatar(src);
    img.onerror = () =>
      message("这张图片打不开，请换一张", { type: "warning" });
    img.src = src;
  };
  reader.readAsDataURL(file);
}

/** 裁剪头像，确定后把裁剪结果（base64）填进表单 */
function cropAvatar(imgSrc: string) {
  const cropRef = ref();
  let avatar = "";
  addDialog({
    title: "裁剪头像",
    width: "40%",
    closeOnClickModal: false,
    fullscreen: deviceDetection(),
    contentRenderer: () =>
      h(ReCropperPreview, {
        ref: cropRef,
        imgSrc,
        canvasOptions: avatarCanvas,
        onCropper: ({ base64 }) => (avatar = base64)
      }),
    beforeSure: done => {
      if (!avatar) return message("请先裁剪头像", { type: "warning" });
      newFormInline.value.avatar = avatar;
      done();
    },
    closeCallBack: () => cropRef.value?.hidePopover()
  });
}

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
    label-width="88px"
  >
    <el-row :gutter="30">
      <!-- 头像 -->
      <re-col :value="24" :xs="24">
        <el-form-item label="头像">
          <div class="flex items-center gap-4">
            <el-avatar
              :size="64"
              :src="avatarPreview || undefined"
              icon="UserFilled"
              class="cursor-pointer border border-dashed border-gray-300"
              @click="() => ($refs.avatarInput as HTMLInputElement).click()"
            />
            <div>
              <el-button
                size="small"
                @click="() => ($refs.avatarInput as HTMLInputElement).click()"
              >
                选择图片
              </el-button>
              <p class="text-xs text-gray-400 mt-1">
                点击头像或按钮选择图片，裁剪后转为 base64 存储
              </p>
            </div>
            <input
              ref="avatarInput"
              type="file"
              accept="image/*"
              class="hidden"
              @change="onAvatarChange"
            />
          </div>
        </el-form-item>
      </re-col>

      <!-- 用户名 -->
      <re-col :value="12" :xs="24" :sm="24">
        <el-form-item label="用户名" prop="username">
          <el-input
            v-model="newFormInline.username"
            clearable
            placeholder="请输入用户名"
            :disabled="!isAdd"
          />
        </el-form-item>
      </re-col>

      <!-- 真实姓名 -->
      <re-col :value="12" :xs="24" :sm="24">
        <el-form-item label="真实姓名" prop="real_name">
          <el-input
            v-model="newFormInline.real_name"
            clearable
            placeholder="请输入真实姓名"
          />
        </el-form-item>
      </re-col>

      <!-- 密码（仅新增时必填） -->
      <re-col v-if="isAdd" :value="12" :xs="24" :sm="24">
        <el-form-item label="密码" prop="password">
          <el-input
            v-model="newFormInline.password"
            clearable
            show-password
            type="password"
            placeholder="请输入密码（留空则使用默认密码）"
          />
        </el-form-item>
      </re-col>

      <!-- 性别 -->
      <re-col :value="12" :xs="24" :sm="24">
        <el-form-item label="性别">
          <el-select
            v-model="newFormInline.gender"
            placeholder="请选择性别"
            class="w-full"
          >
            <el-option label="未设置" :value="-1" />
            <el-option label="男" :value="0" />
            <el-option label="女" :value="1" />
          </el-select>
        </el-form-item>
      </re-col>

      <!-- 手机号 -->
      <re-col :value="12" :xs="24" :sm="24">
        <el-form-item label="手机号" prop="phone">
          <el-input
            v-model="newFormInline.phone"
            clearable
            placeholder="请输入手机号"
          />
        </el-form-item>
      </re-col>

      <!-- 邮箱 -->
      <re-col :value="12" :xs="24" :sm="24">
        <el-form-item label="邮箱" prop="email">
          <el-input
            v-model="newFormInline.email"
            clearable
            placeholder="请输入邮箱"
          />
        </el-form-item>
      </re-col>

      <!-- 角色（多选，实时从数据库） -->
      <re-col :value="12" :xs="24" :sm="24">
        <el-form-item label="分配角色">
          <el-select
            v-model="newFormInline.roles"
            multiple
            clearable
            collapse-tags
            collapse-tags-tooltip
            :max-collapse-tags="3"
            placeholder="请选择角色"
            class="w-full"
          >
            <el-option
              v-for="item in newFormInline.roleOptions"
              :key="item.code"
              :label="item.name"
              :value="item.code"
            />
          </el-select>
        </el-form-item>
      </re-col>

      <!-- 所属部门 -->
      <re-col :value="12" :xs="24" :sm="24">
        <el-form-item label="所属部门">
          <el-cascader
            v-model="newFormInline.dept_id"
            class="w-full"
            :options="newFormInline.deptOptions"
            :props="{
              value: 'id',
              label: 'name',
              emitPath: false,
              checkStrictly: true
            }"
            clearable
            filterable
            placeholder="请选择所属部门（可不选）"
            @clear="newFormInline.dept_id = 0"
          >
            <template #default="{ node, data }">
              <span>{{ data.name }}</span>
              <span v-if="!node.isLeaf"> ({{ data.children.length }}) </span>
            </template>
          </el-cascader>
        </el-form-item>
      </re-col>

      <!-- 状态 -->
      <re-col :value="12" :xs="24" :sm="24">
        <el-form-item label="状态">
          <el-switch
            v-model="newFormInline.status"
            inline-prompt
            :active-value="1"
            :inactive-value="0"
            active-text="启用"
            inactive-text="停用"
            :style="switchStyle"
          />
        </el-form-item>
      </re-col>
    </el-row>
  </el-form>
</template>
