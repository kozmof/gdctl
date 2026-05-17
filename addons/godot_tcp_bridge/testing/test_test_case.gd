@tool
extends "res://addons/godot_tcp_bridge/testing/test_case.gd"

const TestCase = preload("res://addons/godot_tcp_bridge/testing/test_case.gd")


func test_assert_true_passes_without_failures() -> void:
	var subject := TestCase.new()
	subject._gdctl_begin_test("inner")
	subject.assert_true(true)
	var failures := subject._gdctl_end_test()
	assert_eq(failures.size(), 0)


func test_assert_true_records_failure() -> void:
	var subject := TestCase.new()
	subject._gdctl_begin_test("inner")
	subject.assert_true(false, "expected true")
	var failures := subject._gdctl_end_test()
	assert_eq(failures.size(), 1)
	assert_eq(failures[0]["test"], "inner")
	assert_eq(failures[0]["message"], "expected true")


func test_assert_false_passes_without_failures() -> void:
	var subject := TestCase.new()
	subject._gdctl_begin_test("inner")
	subject.assert_false(false)
	var failures := subject._gdctl_end_test()
	assert_eq(failures.size(), 0)


func test_assert_false_records_failure() -> void:
	var subject := TestCase.new()
	subject._gdctl_begin_test("inner")
	subject.assert_false(true, "expected false")
	var failures := subject._gdctl_end_test()
	assert_eq(failures.size(), 1)
	assert_eq(failures[0]["message"], "expected false")


func test_assert_eq_records_default_failure() -> void:
	var subject := TestCase.new()
	subject._gdctl_begin_test("inner")
	subject.assert_eq(1, 2)
	var failures := subject._gdctl_end_test()
	assert_eq(failures.size(), 1)
	assert_true(String(failures[0]["message"]).contains("Expected 1 to equal 2"))


func test_assert_ne_records_default_failure() -> void:
	var subject := TestCase.new()
	subject._gdctl_begin_test("inner")
	subject.assert_ne("same", "same")
	var failures := subject._gdctl_end_test()
	assert_eq(failures.size(), 1)
	assert_true(String(failures[0]["message"]).contains("Expected "))
	assert_true(String(failures[0]["message"]).contains("to not equal"))


func test_assert_eq_and_ne_pass_without_failures() -> void:
	var subject := TestCase.new()
	subject._gdctl_begin_test("inner")
	subject.assert_eq("same", "same")
	subject.assert_ne("left", "right")
	var failures := subject._gdctl_end_test()
	assert_eq(failures.size(), 0)


func test_default_messages_are_used() -> void:
	var subject := TestCase.new()
	subject._gdctl_begin_test("inner")
	subject.assert_true(false)
	subject.assert_false(true)
	var failures := subject._gdctl_end_test()
	assert_eq(failures.size(), 2)
	assert_eq(failures[0]["message"], "Expected value to be true")
	assert_eq(failures[1]["message"], "Expected value to be false")


func test_multiple_failures_keep_order() -> void:
	var subject := TestCase.new()
	subject._gdctl_begin_test("inner")
	subject.fail("first")
	subject.fail("second")
	var failures := subject._gdctl_end_test()
	assert_eq(failures.size(), 2)
	assert_eq(failures[0]["message"], "first")
	assert_eq(failures[1]["message"], "second")


func test_end_test_returns_copy() -> void:
	var subject := TestCase.new()
	subject._gdctl_begin_test("inner")
	subject.fail("first")
	var failures := subject._gdctl_end_test()
	failures[0]["message"] = "changed"
	var failures_again := subject._gdctl_end_test()
	assert_eq(failures_again[0]["message"], "first")


func test_fail_records_message() -> void:
	var subject := TestCase.new()
	subject._gdctl_begin_test("inner")
	subject.fail("boom")
	var failures := subject._gdctl_end_test()
	assert_eq(failures.size(), 1)
	assert_eq(failures[0]["message"], "boom")


func test_begin_test_clears_previous_failures() -> void:
	var subject := TestCase.new()
	subject._gdctl_begin_test("first")
	subject.fail("old")
	subject._gdctl_begin_test("second")
	var failures := subject._gdctl_end_test()
	assert_eq(failures.size(), 0)
