// 角色选项
interface RoleOption {
  id: number;
  name: string;
  code: string;
}

// 新增/编辑用户表单
interface FormItemProps {
  id: number;
  name: string;
  type: number;
  status: number;
  email: string;
  phone: string;
  roles: string[];
  roleOptions: RoleOption[];
  config: string;
  remark: string;
}

// 重置密码表单
interface ResetPwdFormProps {
  id: number;
  password: string;
  pay_password: string;
}

interface FormProps {
  formInline: FormItemProps;
}

interface ResetPwdFormDialogProps {
  formInline: ResetPwdFormProps;
}

export type {
  FormItemProps,
  FormProps,
  ResetPwdFormProps,
  ResetPwdFormDialogProps,
  RoleOption
};
