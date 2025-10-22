# Contributing to HowlerOps

Thank you for your interest in contributing to HowlerOps! We welcome contributions from the community and are excited to work with you.

## Code of Conduct

By participating in this project, you agree to abide by our Code of Conduct:
- Be respectful and inclusive
- Welcome newcomers and help them get started
- Focus on constructive criticism
- Accept feedback gracefully

## How to Contribute

### Reporting Issues

1. Check existing issues to avoid duplicates
2. Use issue templates when available
3. Provide detailed information:
   - Environment (OS, versions)
   - Steps to reproduce
   - Expected vs actual behavior
   - Error messages and logs

### Suggesting Features

1. Check the roadmap and existing feature requests
2. Describe the problem your feature solves
3. Provide use cases and examples
4. Consider implementation complexity

### Contributing Code

#### Setup Development Environment

```bash
# Fork and clone the repository
git clone https://github.com/yourusername/sql-studio.git
cd sql-studio

# Create a feature branch
git checkout -b feature/your-feature-name

# Install dependencies
make deps

# Run tests to ensure everything works
make test
```

#### Development Workflow

1. **Make Changes**
   - Follow existing code style
   - Write meaningful commit messages
   - Add tests for new functionality
   - Update documentation as needed

2. **Test Your Changes**
   ```bash
   # Run unit tests
   make test
   
   # Run linting
   make lint
   
   # Run full validation
   make validate
   ```

3. **CRITICAL: Complete Validation Checklist**
   Before submitting any changes, ensure ALL validation steps pass:
   
   **Frontend Validation:**
   ```bash
   cd frontend
   npm run typecheck    # TypeScript type checking
   npm run lint         # ESLint validation
   npm run test:run     # Unit tests
   ```
   
   **Backend Validation:**
   ```bash
   go mod tidy          # Clean up Go modules
   go fmt ./...         # Format Go code
   go test ./...        # Run Go tests
   ```
   
   **Full Validation:**
   ```bash
   make validate        # Runs lint + test for both frontend and backend
   ```
   
   **Task completion checklist:**
   - [ ] All TypeScript types are valid (`npm run typecheck`)
   - [ ] Frontend code passes linting (`npm run lint`)
   - [ ] Frontend tests pass (`npm run test:run`)
   - [ ] Go modules are tidy (`go mod tidy`)
   - [ ] Go code is formatted (`go fmt ./...`)
   - [ ] Go tests pass (`go test ./...`)
   - [ ] Full validation passes (`make validate`)
   - [ ] Code compiles successfully (`make build`)

4. **Submit Pull Request**
   - Fill out the PR template completely
   - Reference related issues
   - Ensure CI passes
   - Request review from maintainers

### Code Style Guidelines

#### Go Code
- Follow standard Go conventions
- Use `gofmt` and `golint`
- Write descriptive variable names
- Add comments for exported functions
- Keep functions focused and small

#### TypeScript/React Code
- Use TypeScript strict mode
- Follow React hooks best practices
- Use functional components
- Implement proper error boundaries
- Write unit tests with Jest

### Testing Requirements

- Minimum 80% code coverage for new code
- Unit tests for all business logic
- Integration tests for API endpoints
- E2E tests for critical user flows

### Documentation

- Update README for new features
- Add JSDoc/GoDoc comments
- Update API documentation
- Include examples for complex features

## Development Areas

### Database Drivers
Help us add support for new databases:
- Research database-specific features
- Implement driver interface
- Write comprehensive tests
- Document connection parameters

### AI Providers
Expand our AI capabilities:
- Integrate new LLM providers
- Improve prompt engineering
- Optimize context management
- Add specialized models

### Frontend Features
Enhance the user interface:
- Improve query editor
- Add visualization components
- Enhance responsiveness
- Implement accessibility features

### Performance Optimization
Help us go faster:
- Profile and optimize hot paths
- Improve caching strategies
- Optimize database queries
- Reduce memory usage

## Review Process

1. **Initial Review**: Maintainers check for obvious issues
2. **Technical Review**: Deep dive into implementation
3. **Testing**: Verify tests pass and coverage meets standards
4. **Final Approval**: At least two maintainers must approve

## Release Process

- We use semantic versioning (MAJOR.MINOR.PATCH)
- Releases happen bi-weekly (aligned with sprint ends)
- Critical security fixes may trigger immediate releases
- All releases include comprehensive release notes

## Getting Help

- **Discord**: Join our developer channel
- **GitHub Discussions**: Ask questions and share ideas
- **Office Hours**: Weekly sessions with maintainers (Thursdays 3pm UTC)

## Recognition

We value all contributions:
- Contributors are listed in CONTRIBUTORS.md
- Significant contributions are highlighted in release notes
- Top contributors get special Discord roles
- Annual contributor appreciation events

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

---

Thank you for helping make HowlerOps better! 🌟