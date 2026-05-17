@tool
extends "res://addons/godot_tcp_bridge/testing/test_case.gd"

var before_all_count := 0
var before_each_count := 0
var after_each_count := 0
var order: Array[String] = []


func before_all() -> void:
	before_all_count += 1
	order.append("before_all")


func before_each() -> void:
	before_each_count += 1
	order.append("before_each_%d" % before_each_count)


func after_each() -> void:
	after_each_count += 1
	order.append("after_each_%d" % after_each_count)


func after_all() -> void:
	assert_eq(before_all_count, 1)
	assert_eq(before_each_count, 2)
	assert_eq(after_each_count, 2)
	assert_eq(order, [
		"before_all",
		"before_each_1",
		"test_alpha",
		"after_each_1",
		"before_each_2",
		"test_beta",
		"after_each_2",
	])


func test_alpha() -> void:
	order.append("test_alpha")
	assert_eq(before_all_count, 1)
	assert_eq(before_each_count, 1)
	assert_eq(after_each_count, 0)


func test_beta() -> void:
	order.append("test_beta")
	assert_eq(before_all_count, 1)
	assert_eq(before_each_count, 2)
	assert_eq(after_each_count, 1)
