import type { FormRules } from "element-plus";

// 新增用户校验规则
export const formRules: FormRules = {
  name: [
    { required: true, message: "请输入用户名称", trigger: "blur" },
    { min: 1, max: 128, message: "用户名称长度为1-128个字符", trigger: "blur" }
  ],
  type: [
    { required: true, message: "请选择用户类型", trigger: "change" },
    {
      type: "number",
      min: 1,
      message: "请选择用户类型",
      trigger: "change"
    }
  ],
  status: [{ required: true, message: "请选择用户状态", trigger: "change" }],
  email: [
    { required: true, message: "请输入用户邮箱", trigger: "blur" },
    { type: "email", message: "请输入正确的邮箱格式", trigger: "blur" },
    { max: 128, message: "邮箱长度不超过128个字符", trigger: "blur" }
  ],
  phone: [
    { required: true, message: "请输入用户手机", trigger: "blur" },
    { max: 20, message: "手机号长度不超过20个字符", trigger: "blur" }
  ]
};

// 编辑用户校验规则（无需密码）
export const editFormRules: FormRules = {
  name: formRules.name,
  type: formRules.type,
  status: formRules.status,
  email: formRules.email,
  phone: formRules.phone
};

// 重置密码校验规则
export const resetPwdRules: FormRules = {
  password: [
    { min: 6, max: 255, message: "密码长度为6-255个字符", trigger: "blur" }
  ],
  pay_password: [
    { min: 6, max: 255, message: "密码长度为6-255个字符", trigger: "blur" }
  ]
};
