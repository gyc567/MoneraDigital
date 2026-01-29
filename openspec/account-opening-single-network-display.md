# AccountOpening Single Network Display

## Issue Summary

在 `/dashboard/account-opening` 页面，当前实现中：
- 多网络时：显示网络切换Tab + 钱包ID + 地址
- 单网络时：不显示网络标签 + 钱包ID + 地址

**需求**：单网络时，也要在钱包ID上方显示当前网络标签。

## Current Implementation

**多网络显示逻辑** (AccountOpening.tsx):
```tsx
{availableNetworks.length > 1 && (
  <Tabs value={selectedNetwork} onValueChange={setSelectedNetwork}>
    <TabsList>...</TabsList>
  </Tabs>
)}
```

**问题**：单网络时没有网络标识，用户无法快速确认当前地址属于哪个网络。

## Proposed Solution

### 核心设计原则

1. **KISS (Keep It Simple, Stupid)**
   - 单网络时显示静态网络标签（非Tab）
   - 复用现有样式和布局
   - 最小改动

2. **高内聚，低耦合**
   - 网络显示逻辑内聚在成功状态组件内
   - 不影响其他功能模块
   - 独立的测试覆盖

3. **100% 测试覆盖率**
   - 所有新增代码路径都有测试

### 实现方案

#### 1. 新增静态网络标签组件

```tsx
// 单网络时显示的网络标签
{availableNetworks.length === 1 && (
  <div className="flex items-center gap-2 px-3 py-2 rounded-md bg-secondary/50 border border-border">
    <Globe className="w-4 h-4 text-muted-foreground" />
    <span className="text-sm font-medium">
      {availableNetworks[0].label}
    </span>
  </div>
)}
```

#### 2. 统一网络显示逻辑

```tsx
// 网络显示（Tab或多网络时，单标签单网络时）
{availableNetworks.length > 0 && (
  <>
    {availableNetworks.length > 1 ? (
      <Tabs value={selectedNetwork} onValueChange={setSelectedNetwork}>
        <TabsList>...</TabsList>
      </Tabs>
    ) : (
      <div className="flex items-center gap-2 px-3 py-2 rounded-md bg-secondary/50 border border-border">
        <Globe className="w-4 h-4 text-muted-foreground" />
        <span className="text-sm font-medium">
          {availableNetworks[0].label}
        </span>
      </div>
    )}
  </>
)}
```

### 文件变更

1. **`src/pages/dashboard/AccountOpening.tsx`**
   - 修改网络显示逻辑：单网络时显示静态标签

2. **`src/pages/dashboard/AccountOpening.test.tsx`**
   - 新增单网络显示测试
   - 新增多网络Tab显示测试
   - 新增网络标签样式测试

### 测试用例

1. `should show network label when single network available`
   - 单网络时显示网络标签
   - 标签显示正确的网络名称
   - 包含Globe图标

2. `should show network tabs when multiple networks available`
   - 多网络时显示Tab（现有测试增强）

3. `should display correct network label for each network`
   - TRON网络显示"TRON"
   - ETH网络显示"ETH"
   - BSC网络显示"BSC"

### 风险评估

- **风险级别**: 极低
- **影响范围**: 仅AccountOpening页面成功状态的UI显示
- **回滚方案**: 简单恢复原代码即可

### 验证标准

- [ ] 代码符合KISS原则
- [ ] 高内聚低耦合设计
- [ ] 100%测试覆盖率
- [ ] 无回归影响
- [ ] 构建成功
- [ ] Lint通过
- [ ] 所有测试通过
