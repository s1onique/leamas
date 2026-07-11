# GitHub Copilot Instructions

<!-- LEAMAS:EXECUTABLE-CONTRACT-FIRST:BEGIN -->
## Executable Contract First

For every behavior-changing task:

1. Inspect the existing behavioral contract and relevant tests.
2. Before editing production code, identify the narrowest stable boundary
   and design an orthogonal, declarative test matrix.
3. Implement the relevant tests and run them to establish RED for the
   intended behavioral reason.
4. Only then implement the smallest coherent production change.
5. Establish focused GREEN, run affected subsystem tests, and run the
   repository gate.
6. Refactor only while the executable contract remains green.

Test observable behavior rather than private implementation details.
Prefer table-driven tests where cases share execution logic. Keep tests
deterministic and explicit. Prefer injected capabilities or simple fakes
over interaction-heavy mocks. Do not weaken a correct test merely to make
an implementation pass. Document any exception to the RED requirement.
<!-- LEAMAS:EXECUTABLE-CONTRACT-FIRST:END -->
