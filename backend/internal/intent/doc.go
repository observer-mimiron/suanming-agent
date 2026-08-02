// Package intent 提供面向用户消息的共享 lexical detector。
//
// 本包是 birth-info / explicit method / explicit action / timing 检测的唯一 truth source，
// 供 supervisor 和 runtime 共同使用，消除重复实现导致的 split-brain。
package intent
