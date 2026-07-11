package main

// registerActDoctrineCompiler is a placeholder for ACT-local
// registration hooks. The current registration is performed via the
// explicit `case "act-doctrine-compiler":` branch in
// handleFactoryVerify, so this function is a no-op kept here to make
// the registration surface explicit and grep-able.
//
// Future ACTs may use this hook to register additional checks.
func registerActDoctrineCompiler() {}
