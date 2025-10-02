# Development Guidelines

Please follow the instructions in the README and `.github/copilot-instructions.md` for all development workflow, build, lint, and test steps.

Before submitting changes, run at minimum:

```bash
make lint        # Lint both backend and frontend
make generate    # Generate protobuf code
make be.build    # Build backend
make fe.build    # Build frontend
make test        # Run backend tests
```

For more details, see the [README](README.md) and run `make help` for all available commands.
