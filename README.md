# Mailings

Newsletter- und Mailing-Anwendung der Nakami Lounge.

Siehe [CLAUDE.md](./CLAUDE.md) für Architektur und Setup.

## Quickstart

```bash
git submodule update --init --recursive
make up
make bob
go run . createUser -s Herr -e admin@example.com -f Admin -l User -p Hawaii11 -a -d
task dev
```
