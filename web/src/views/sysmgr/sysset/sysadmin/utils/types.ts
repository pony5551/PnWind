export interface RoleOption {
  id: number;
  name: string;
  code: string;
}

export interface FormItemProps {
  id?: number;
  title: string; // "新增" | "修改"
  username: string;
  real_name: string;
  password: string;
  gender: number; // 0男 1女 -1未设置
  phone: string;
  email: string;
  status: number; // 0停用 1启用
  roles: string[]; // 角色code数组
  avatar: string; // base64 头像
  dept_id: number; // 所属部门id，0=未设置
  deptOptions?: any[];
  roleOptions?: RoleOption[];
}

export interface FormProps {
  formInline: FormItemProps;
}

export interface RoleFormItemProps {
  id: number;
  username: string;
  real_name: string;
  roles: string[]; // 已选角色code数组
  roleOptions: RoleOption[];
}

export interface RoleFormProps {
  formInline: RoleFormItemProps;
}

// 兼容旧导出名
export type { FormItemProps as default };
