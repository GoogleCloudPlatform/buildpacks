from typing import Any

class NodeJSExample:
    def __init__(self):
        self.ctx = None
        self.yarn_layer = None
        self.pjs = None

    def example_maker_yarn_installer(self) -> None:
        """
        Example of installing Yarn using the MakerYarnInstaller capability.
        """
        if self.ctx is not None and hasattr(self.ctx, 'Capability'):
            cap = self.ctx.Capability('yarn-installer')
            if isinstance(cap, NodeJSYarnInstaller):
                cap.InstallYarn(self.ctx, self.yarn_layer, self.pjs)

    def example_maker_yarn1_module_installer(self) -> None:
        """
        Example of installing Yarn 1 modules using the MakerYarn1ModuleInstaller capability.
        """
        if self.ctx is not None and hasattr(self.ctx, 'Capability'):
            cap = self.ctx.Capability('yarn1-module-installer')
            if isinstance(cap, NodeJSYarn1ModuleInstaller):
                cap.InstallModules(self.ctx, self.pjs)


class NodeJSYarnInstaller:
    def __init__(self, ctx: Any) -> None:
        """
        Yarn installer capability.
        """
        self.ctx = ctx

    def InstallYarn(self, ctx: Any, yarn_layer: Any, pjs: Any) -> None:
        pass


class NodeJSYarn1ModuleInstaller:
    def __init__(self, ctx: Any) -> None:
        """
        Yarn 1 module installer capability.
        """
        self.ctx = ctx

    def InstallModules(self, ctx: Any, pjs: Any) -> None:
        pass
