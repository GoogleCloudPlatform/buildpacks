import unittest
from typing import List, Tuple

from gcp_buildpack.builder_context import BuilderContext
from gcp_buildpack.buildpack_info import BuildpackInfo
from gcp_buildpack.detect_context import DetectContext
from gcp_buildpack.fake_exit_handler import FakeExitHandler
from gcp_buildpack.process_option import ProcessOption
from gcp_buildpack.processes import Processes

class TestGCPBuildpack(unittest.TestCase):
    def test_debug_mode_initialized(self):
        test_cases: List[Tuple[str, str]] = [
            ("",),
            ("true", True),
            ("false", False),
        ]
        for tc in test_cases:
            if tc[1]:
                with self.subTest(name=tc[0]):
                    BuilderContext.debug_mode = "true"
                    context = BuilderContext()
                    self.assertTrue(context.debug)
                    self.assertTrue(context.Debug())
            else:
                with self.subTest(name=tc[0]):
                    delattr(BuilderContext, 'debug_mode')
                    context = BuilderContext()
                    self.assertFalse(context.debug)
                    self.assertFalse(context.Debug())

    def test_new_context_with_application_root(self):
        application_root = "myroot"
        context = BuilderContext(application_root=application_root)
        self.assertEqual(context.application_root, application_root)

    def test_new_context_with_buildpack_info(self):
        buildpack_info = BuildpackInfo(name="myname")
        context = BuilderContext(buildpack_info=buildpack_info)
        self.assertEqual(context.info.name, buildpack_info.name)

    def test_detect_context_initialized(self):
        id = "my-id"
        version = "my-version"
        name = "my-name"

        detect_handler = FakeExitHandler()
        DetectContext(detect_handler=detect_handler).detect(lambda context: OptIn("some reason"), libcnb.WithExitHandler(detect_handler))
        self.assertEqual(context.BuildpackID(), id)
        self.assertEqual(context.BuildpackVersion(), version)
        self.assertEqual(context.BuildpackName(), name)

    def test_detect_emits_span(self):
        span = DetectContext().detect(lambda context: OptIn("some reason"), libcnb.WithExitHandler(FakeExitHandler()))[0]
        self.assertEqual(span.name, "Buildpack Detect")
        self.assertTrue(span.start.is_set)
        self.assertTrue(span.end.after(span.start))
        self.assertEqual(span.status, buildererror.StatusOk)

    def test_build_context_initialized(self):
        id = "my-id"
        version = "my-version"
        name = "my-name"

        build_handler = FakeExitHandler()
        context = BuilderContext(build_handler=build_handler).build(lambda context: None)
        self.assertEqual(context.BuildpackID(), id)
        self.assertEqual(context.BuildpackVersion(), version)
        self.assertEqual(context.BuildpackName(), name)

    def test_build_emits_span(self):
        span = BuilderContext().build(lambda context: None)[0]
        self.assertEqual(span.name, "Buildpack Build")
        self.assertTrue(span.start.is_set)
        self.assertTrue(span.end.after(span.start))
        self.assertEqual(span.status, buildererror.StatusOk)

    def test_add_web_process(self):
        ctx = BuilderContext()
        ctx.add_web_process(["/start"])
        expected_processes: List[Processes] = [Processes(command=["bash", "-c", "/start"], type="web")]
        self.assertEqual(ctx.build_result.processes, expected_processes)

    def test_add_process(self):
        test_cases: List[Tuple[str, str]] = [
            ("web", "single"),
            ("dev", "one two"),
            ("cli", "invoke $hello"),
            ("web", "invoke >pipe"),
            ("dev", "invoke --flag && another"),
            ("cli", "echo \"DOUBLE QUOTES\""),
            ("web", "echo 'SINGLE QUOTES'"),
            ("dev", "test characters"),
        ]
        for tc in test_cases:
            with self.subTest(name=tc[0]):
                ctx = BuilderContext()
                ctx.add_process(tc[0], tc[1])
                expected_processes: List[Processes] = [Processes(command=[tc[1]], type=tc[0])]
                self.assertEqual(ctx.build_result.processes, expected_processes)

    def test_add_label(self):
        key_value_pairs := ["my-key=my-value"]
        ctx = BuilderContext()
        for pair in key_value_pairs:
            key, value = pair.split("=")
            ctx.add_label(key, value)
        self.assertEqual(ctx.build_result.labels, [{"key": "google.my-key", "value": "my-value"}])

    def test_add_label_errors(self):
        invalid_key_values := ["", "0", "00invalid", "abc def", "abd@def", "  abc", "def  ", "a__b"]
        for key_value in invalid_key_values:
            ctx = BuilderContext()
            ctx.add_label(key_value, "some-value")
            self.assertEqual(ctx.build_result.labels, [])

    def test_has_at_least_one(self):
        patterns := ["*.py"]
        for pattern in patterns:
            with self.subTest(name=pattern):
                context, _ = simple_context(self)
                files := ["/file.py", "/node_modules/file.rb"]
                context.application_root = "/"
                for file in files:
                    _, err := os.Create(file)
                    if err != nil:
                        raise
                found, err := context.has_at_least_one(pattern)
                self.assertTrue(found)
                self.assertIsNone(err)

    def test_has_at_least_one_filtered(self):
        patterns := ["*.py"]
        for pattern in patterns:
            with self.subTest(name=pattern):
                context, _ = simple_context(self)
                files := ["/file.py", "/node_modules/file.rb"]
                context.application_root = "/"
                for file in files:
                    _, err := os.Create(file)
                    if err != nil:
                        raise
                found, err := context.has_at_least_one_filtered(pattern, lambda path: True)
                self.assertTrue(found)
                self.assertIsNone(err)

    def test_has_at_least_one_outside_dependency_directories(self):
        patterns := ["*.py"]
        for pattern in patterns:
            with self.subTest(name=pattern):
                context, _ = simple_context(self)
                files := ["/file.py", "/node_modules/file.rb"]
                context.application_root = "/"
                for file in files:
                    _, err := os.Create(file)
                    if err != nil:
                        raise
                found, err := context.has_at_least_one_outside_dependency_directories(pattern)
                self.assertFalse(found)
                self.assertIsNone(err)

    def test_disabled_capabilities(self):
        ctx = BuilderContext(
            with_disabled_capability="cap1",
            with_disabled_capability="cap2"
        )
        self.assertTrue(ctx.is_disabled("cap1"))
        self.assertTrue(ctx.is_disabled("cap2"))
        self.assertFalse(ctx.is_disabled("cap3"))

class simple_context(unittest.TestCase):
    def setUp(self):
        self.context, _ = BuilderContext(simple_context=self)

def setOSArgs(args: List[str]) -> None:
    os.environ["PATH"] = ":".join(args)
