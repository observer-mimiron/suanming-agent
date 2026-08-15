// 本文件属于紫微 domain 层。
// 本文件负责表达星曜及其领域属性。
// 不负责命盘组装、工具参数、Session、模型、trace、SSE 或最终文本。
package domain

// ZiWeiStar 表示紫微斗数中的一颗星曜及其领域属性。
// Type 区分星曜类别，Brightness 表示庙旺利得平陷，Mutagen 表示四化结果。
type ZiWeiStar struct {
	Name       string
	Type       string
	Brightness string
	Mutagen    string
}
