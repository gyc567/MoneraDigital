# AccountOpening Network Tabs Feature

## Issue Summary

在 `/dashboard/account-opening` 页面，需要在钱包ID上方增加一个显示网络的tab（如 TRON, ETH, BSC 等），让用户可以切换查看不同网络的地址。

## Current Implementation

当前 `AccountOpening.tsx` 在成功状态下：
- 显示 `walletId`（钱包ID）
- 显示单一 `displayAddress`（地址）
- 通过 `addresses` JSON 解析出多个链的地址，但只显示选中的 `selectedCurrency` 对应的地址

**当前 addresses 格式：**
```json
{"TRON":"TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW","ETH":"0x71C7656EC7ab88b098defB751B7401B5f6d8976F","BSC":"..."}
```

## Proposed Solution

### 核心设计原则

1. **KISS (Keep It Simple, Stupid)**
   - 最小改动，只添加必要的状态和UI
   - 复用现有的 Tab 组件
   - 保持代码简洁

2. **高内聚，低耦合**
   - 网络解析逻辑内聚在组件内
   - 不影响其他功能模块
   - 独立的测试覆盖

3. **100% 测试覆盖率**
   - 所有新增代码路径都有测试

### 实现方案

#### 1. 新增类型定义

```typescript
// 网络选项类型
interface NetworkOption {
  value: string;  // "TRON", "ETH", "BSC"
  label: string;  // 显示名称
}

// 可用网络列表（从 addresses JSON 解析）
const getAvailableNetworks = (addressesJson: string): NetworkOption[] => {
  try {
    const addrMap = JSON.parse(addressesJson);
    return Object.keys(addrMap).map(network => ({
      value: network,
      label: network  // 简化显示，直接使用网络代码
    }));
  } catch {
    return [];
  }
};
```

#### 2. 新增状态

```typescript
const [selectedNetwork, setSelectedNetwork] = useState<string>("");
const [availableNetworks, setAvailableNetworks] = useState<NetworkOption[]>([]);
```

#### 3. UI 变更

在钱包ID显示区域上方添加网络切换 Tab：

```tsx
{/* Network Tabs - 在 walletId 上方 */}
{availableNetworks.length > 1 && (
  <Tabs value={selectedNetwork} onValueChange={setSelectedNetwork}>
    <TabsList className="grid w-full" style={{ gridTemplateColumns: `repeat(${availableNetworks.length}, minmax(0, 1fr))` }}>
      {availableNetworks.map(network => (
        <TabsTrigger key={network.value} value={network.value}>
          {network.label}
        </TabsTrigger>
      ))}
    </TabsList>
  </Tabs>
)}
```

#### 4. 地址显示逻辑

```typescript
// 根据选中的网络显示对应地址
const displayAddress = addrMap[selectedNetwork] || "";
```

### 文件变更

1. **`src/pages/dashboard/AccountOpening.tsx`**
   - 新增 `NetworkOption` 类型
   - 新增 `getAvailableNetworks` 工具函数
   - 新增 `selectedNetwork` 和 `availableNetworks` 状态
   - 在钱包ID上方添加网络切换 Tab
   - 更新地址显示逻辑

2. **`src/pages/dashboard/AccountOpening.test.tsx`**
   - 新增网络解析测试
   - 新增 Tab 切换测试
   - 新增多网络地址显示测试
   - 新增单网络场景测试

### 测试用例

1. `getAvailableNetworks` 解析测试
   - ✓ 正确解析多个网络
   - ✓ 处理空字符串
   - ✓ 处理无效 JSON
   - ✓ 返回正确格式

2. Tab UI 渲染测试
   - ✓ 单网络时不显示 Tab
   - ✓ 多网络时显示所有 Tab
   - ✓ Tab 数量正确

3. Tab 切换功能测试
   - ✓ 切换网络更新显示地址
   - ✓ 默认选中第一个网络
   - ✓ 地址正确显示

4. 集成测试
   - ✓ 完整成功流程测试
   - ✓ 多网络场景验证
   - ✓ 无回归影响现有功能

### 风险评估

- **风险级别**: 低
- **影响范围**: 仅限 AccountOpening 页面成功状态
- **回滚方案**: 简单移除新增代码即可恢复

### 验证标准

- [ ] 代码符合 KISS 原则
- [ ] 高内聚低耦合设计
- [ ] 100% 测试覆盖率
- [ ] 无回归影响
- [ ] 构建成功
- [ ] Lint 通过
- [ ] 所有测试通过
