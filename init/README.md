# systemd

Change `WorkingDirectory` and `ExecStart` paths and put this services at `/lib/systemd/system/`.

To start core execute

```bash
$ sudo systemctl start ric-edge-core.service
```

To start connector execute

```bash
$ sudo systemctl start ric-edge@<connector>.service
```

To view core logs execute

```bash
$ sudo journalctl -u ric-edge-core.service
```

To view connector logs execute

```bash
$ sudo journalctl -u ric-edge@<connector>.service
```
