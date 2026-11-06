class CoreEngine:
    """
    A core component that represents the high-level business logic.
    """
    def __init__(self, config: dict):
        self.config = config
        self._is_running = False  # Private attribute

    def run(self):
        self._is_running = True
        print(f"[Core] Engine starting with config: {self.config}")
        import time
        time.sleep(0.1)
        print("[Core] Engine is now running.")

    def stop(self):
        self._is_running = False
        print("[Core] Engine stopped.")

    @property
    def status(self):
        return "RUNNING" if self._is_running else "STOPPED"
