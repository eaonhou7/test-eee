package amazon

import "github.com/flipped-aurora/gin-vue-admin/server/global"

type FinanceQuestion struct {
	global.GVA_MODEL
	Title        string `json:"title" gorm:"column:title;type:varchar(255);index;comment:问题标题"`
	QuestionType string `json:"questionType" gorm:"column:question_type;type:varchar(64);index;comment:问题类型"`
	ContentHTML  string `json:"contentHtml" gorm:"column:content_html;type:longtext;comment:问题内容"`
}

func (FinanceQuestion) TableName() string {
	return "amazon_finance_questions"
}

type FinanceQuestionType struct {
	global.GVA_MODEL
	Name string `json:"name" gorm:"column:name;type:varchar(64);uniqueIndex;comment:问题类型"`
	Sort int    `json:"sort" gorm:"column:sort;index;default:0;comment:排序"`
}

func (FinanceQuestionType) TableName() string {
	return "amazon_finance_question_types"
}
