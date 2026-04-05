import main


def test_main_uses_cli_host_and_port(monkeypatch):
    captured: dict[str, object] = {}

    def fake_run(app, host, port):
        captured["app"] = app
        captured["host"] = host
        captured["port"] = port

    monkeypatch.setattr(main.uvicorn, "run", fake_run)

    main.main(["--host", "0.0.0.0", "--port", "9012"])

    assert captured == {"app": main.app, "host": "0.0.0.0", "port": 9012}


def test_main_restores_missing_console_streams(monkeypatch):
    captured: dict[str, object] = {}

    def fake_run(app, host, port):
        captured["stdout"] = main.sys.stdout
        captured["stderr"] = main.sys.stderr

    monkeypatch.setattr(main.uvicorn, "run", fake_run)
    monkeypatch.setattr(main.sys, "stdout", None)
    monkeypatch.setattr(main.sys, "stderr", None)

    main.main([])

    assert captured["stdout"] is not None
    assert captured["stderr"] is not None
