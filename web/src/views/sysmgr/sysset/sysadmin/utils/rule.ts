import { reactive } from "vue";
import type { FormRules } from "element-plus";
import { isPhone, isEmail } from "@pureadmin/utils";

export const formRules = reactive(<FormRules>{
  username: [{ required: true, message: "用户名称为必填项", trigger: "blur" }],
  password: [{ required: true, message: "用户密码为必填项", trigger: "blur" }],
  phone: [
    {
      validator: (rule, value, callback) => {
        if (!value) {
          callback();
          return;
        }
        if (!isPhone(value)) callback(new Error("请输入正确的手机号码格式"));
        else callback();
      },
      trigger: "blur"
    }
  ],
  email: [
    {
      validator: (rule, value, callback) => {
        if (!value) {
          callback();
          return;
        }
        if (!isEmail(value)) callback(new Error("请输入正确的邮箱格式"));
        else callback();
      },
      trigger: "blur"
    }
  ]
});
