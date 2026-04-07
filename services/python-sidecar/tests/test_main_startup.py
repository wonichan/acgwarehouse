import main
from fastapi.testclient import TestClient


def test_main_uses_cli_host_and_port(monkeypatch):
    captured: dict[str, object] = {}

    def fake_run(app, host, port, **kwargs):
        captured["app"] = app
        captured["host"] = host
        captured["port"] = port
        captured.update(kwargs)

    monkeypatch.setattr(main.uvicorn, "run", fake_run)

    main.main(["--host", "0.0.0.0", "--port", "9012"])

    assert captured == {"app": main.app, "host": "0.0.0.0", "port": 9012}


def test_main_restores_missing_console_streams(monkeypatch):
    captured: dict[str, object] = {}

    def fake_run(app, host, port, **kwargs):
        captured["stdout"] = main.sys.stdout
        captured["stderr"] = main.sys.stderr

    monkeypatch.setattr(main.uvicorn, "run", fake_run)
    monkeypatch.setattr(main.sys, "stdout", None)
    monkeypatch.setattr(main.sys, "stderr", None)

    main.main([])

    assert captured["stdout"] is not None
    assert captured["stderr"] is not None


def test_shutdown_endpoint_returns_accepted_and_schedules_shutdown(monkeypatch):
    client = TestClient(main.app)
    called = {"count": 0, "delay": None}

    def fake_schedule_shutdown(delay_seconds: float = 0.05):
        called["count"] += 1
        called["delay"] = delay_seconds

    monkeypatch.setattr(main, "schedule_shutdown", fake_schedule_shutdown)

    response = client.post("/shutdown")

    assert response.status_code == 202
    assert response.json() == {"status": "shutting_down"}
    assert called["count"] == 1
    assert called["delay"] == 1.0
