package models

// DataIdForm 通用单 ID 参数（用于需要单条记录 ID 的操作，如查询详情）
type DataIdForm struct {
	Id int64 `db:"id" json:"id" validate:"required,gt=0" label:"id"`
}

// DataIdsForm 通用批量 ID 参数（用于批量删除等操作）
type DataIdsForm struct {
	Ids []int64 `db:"ids" json:"ids" validate:"required,min=1,dive,gt=0" label:"ids"`
}

// PageParams 分页请求基础参数，可嵌入到各功能的 SearchParams 中
//
//	type XxxSearchParams struct {
//	    Name string `json:"name" default:""`
//	    models.PageParams
//	}
type PageParams struct {
	PageSize    uint64 `json:"pageSize"    default:"20"`
	CurrentPage uint64 `json:"currentPage" default:"1"`
}

// PageData 分页响应基础结构，可嵌入到各功能的 ListData 中
//
//	type XxxListData struct {
//	    List []XxxListItem `json:"list"`
//	    models.PageData
//	}
type PageData struct {
	Total       uint64 `json:"total"`
	PageSize    uint64 `json:"pageSize"`
	CurrentPage uint64 `json:"currentPage"`
}
