# Go Guardian

手动创建 GitHub 仓库步骤：

1. **访问 GitHub**
   ```
   https://github.com/new
   ```

2. **创建仓库**
   - Repository name: `go-guardian`
   - Description: `Production-ready middleware toolkit for Go: rate limiting, circuit breaking, tracing, metrics. Works with HTTP, gRPC, and any Go service.`
   - Public
   - 不要初始化 README、.gitignore 或 License（本地已有）

3. **推送代码**
   ```bash
   cd /Users/wanghuayang/code/bcb/agent/go-guardian
   git remote add origin https://github.com/why19970628/go-guardian.git
   git branch -M main
   git push -u origin main
   git push --tags
   ```

4. **创建第一个 Release**
   - 访问 `https://github.com/why19970628/go-guardian/releases/new`
   - Tag: `v0.1.0`
   - Title: `v0.1.0 - Initial Release`
   - Description:
   ```
   ## 🎉 Initial Release
   
   Production-ready middleware toolkit for Go applications.
   
   ### Core Packages
   - ✅ Rate Limiting (token bucket + Redis)
   - ✅ Circuit Breaker (three-state)
   - ✅ Distributed Tracing (trace ID generation & propagation)
   - ✅ Prometheus Metrics (QPS, latency, errors)
   - ✅ Structured Logger (Zap + trace context)
   
   ### Framework Adapters
   - ✅ Gin middleware
   - ✅ gRPC interceptors
   
   ### Extensions
   - ✅ LLM metrics & fallback strategies
   - ✅ Eino framework integration
   
   ### Documentation
   - 📚 Complete README (English + 中文)
   - 📚 Quick Start Guide
   - 📚 Contributing Guidelines
   - 📚 Example projects
   
   ### Installation
   ```bash
   go get github.com/why19970628/go-guardian/ratelimit
   go get github.com/why19970628/go-guardian/circuitbreaker
   go get github.com/why19970628/go-guardian/trace
   go get github.com/why19970628/go-guardian/metrics
   ```
   ```

5. **完成后的 URL**
   - 仓库: `https://github.com/why19970628/go-guardian`
   - Release: `https://github.com/why19970628/go-guardian/releases/tag/v0.1.0`
