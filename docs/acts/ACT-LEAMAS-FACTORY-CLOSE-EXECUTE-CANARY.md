# ACT-LEAMAS-FACTORY-CLOSE-EXECUTE-CANARY

## ID

ACT-LEAMAS-FACTORY-CLOSE-EXECUTE-CANARY

## BASE

```text
BASE=<freeze revision supplied by Closure Factory>
```

## MISSION

Change one deterministic fixture so the canary ACT proves the
new `factory close execute` command works end-to-end.

## ACCEPTANCE

Run one named test:

```text
go test -count=1 -run TestCanaryFixture ./internal/factory/closure/...
```

## PUBLICATION

One subject commit.

## Notes

This ACT demonstrates a coding ACT consumer:

- It contains NO literal self/future commit OID.
- It contains NO lifecycle choreography.
- It contains NO gate-output parsing shell.
- It contains NO evidence JSON shell.
- It contains NO manual report commit.
- It contains NO manual annotated-tag construction.

All identities, choreography, evidence collection, and tag
creation are produced by `leamas factory close execute`.
