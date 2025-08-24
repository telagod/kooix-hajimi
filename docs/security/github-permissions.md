# GitHub权限配置

## 权限级别说明

Kooix Hajimi支持不同级别的GitHub权限，根据你的使用需求选择合适的权限级别。

## 基础权限（仅扫描功能）

**所需权限**:
- ✅ `public_repo` - 搜索公共仓库代码
- ✅ `read:user` - 读取用户信息（用于API配额管理）

**适用场景**:
- 仅需要扫描和发现泄露的API密钥
- 不需要自动创建安全警告
- 开发和测试环境

**Token创建步骤**:
1. 访问 https://github.com/settings/tokens
2. 点击"Generate new token (classic)"
3. 选择以下权限：
   - ✅ `public_repo`
   - ✅ `read:user`
4. 生成并保存token

## 完整权限（包含安全通知）

**所需权限**:
- ⚠️ `repo` - 完整仓库访问权限
- ✅ `read:user` - 读取用户信息

**适用场景**:
- 需要在发现密钥泄露时自动创建GitHub issue
- 主动安全通知和漏洞披露
- 生产环境安全监控

**重要警告**:
🚨 `repo`权限提供对所有仓库的完整访问权限，包括私有仓库。请确保：
- 只在受信任的环境中使用
- 定期轮换token
- 使用专用的GitHub账户

**Token创建步骤**:
1. 访问 https://github.com/settings/tokens
2. 点击"Generate new token (classic)"
3. 选择以下权限：
   - ✅ `repo` (完整仓库访问)
   - ✅ `read:user`
4. 生成并保存token

## 权限配置策略

### 开发环境
```yaml
# 使用基础权限
security_notifications:
  enabled: true
  create_issues: false  # 不创建真实issue
  dry_run: true        # 启用测试模式
```

### 测试环境
```yaml
# 使用完整权限，但启用干运行
security_notifications:
  enabled: true
  create_issues: true
  dry_run: true        # 测试模式，不创建真实issue
  notify_on_severity: "critical"  # 只测试最高级别
```

### 生产环境
```yaml
# 使用完整权限，谨慎配置
security_notifications:
  enabled: true
  create_issues: true
  dry_run: false
  notify_on_severity: "high"  # 高级别及以上
```

## 安全建议

### Token管理
1. **专用账户**: 为Kooix Hajimi创建专用的GitHub账户
2. **权限最小化**: 根据实际需求选择最小权限
3. **定期轮换**: 定期更新和轮换tokens
4. **环境隔离**: 不同环境使用不同的tokens

### 监控和审计
1. **使用记录**: GitHub提供token使用记录和审计日志
2. **权限审查**: 定期审查token权限和使用情况
3. **异常检测**: 监控token的异常使用行为

### 权限验证
创建token后，可以通过以下方式验证权限：

```bash
# 测试基础权限
curl -H "Authorization: token YOUR_TOKEN" https://api.github.com/user

# 测试搜索权限
curl -H "Authorization: token YOUR_TOKEN" \
  "https://api.github.com/search/code?q=AIzaSy+language:python"

# 测试issue创建权限（仅限完整权限token）
curl -X POST \
  -H "Authorization: token YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title":"Test Issue","body":"Test"}' \
  https://api.github.com/repos/YOUR_USERNAME/test-repo/issues
```

## 严重级别和通知策略

### 密钥类型严重级别
- 🔴 **Critical**: AWS、GitHub、Stripe等高风险服务
- 🟠 **High**: OpenAI、Gemini、Claude等AI服务
- 🟡 **Medium**: 其他API服务

### 推荐通知策略
- **Conservative**: 仅Critical级别 (`notify_on_severity: "critical"`)
- **Balanced**: High及以上级别 (`notify_on_severity: "high"`)
- **Aggressive**: 所有级别 (`notify_on_severity: "all"`)

## 故障排除

### 常见权限错误
1. **403 Forbidden**: Token权限不足或已过期
2. **404 Not Found**: Token无法访问指定仓库
3. **422 Unprocessable Entity**: Issue创建参数错误

### 权限测试
使用Web界面的"干运行"模式测试配置：
1. 启用 `dry_run: true`
2. 配置适当的severity级别
3. 运行测试扫描
4. 查看日志确认配置正确