# Add New Wallet Address Feature

## Issue Summary

用户 `23132@112.com` (user ID 73) 访问 `/dashboard/account-opening` 页面时，需要：
1. 增加"新增账户地址"按钮
2. 允许用户开通其他网络的充值地址
3. 增加代币名称（USDC, USDT）下拉菜单选项

## Current Implementation

当前 `AccountOpening.tsx` 只支持创建钱包，没有后续添加地址功能：
- 成功创建钱包后显示现有地址
- 只显示一个货币选项（selectedCurrency）
- 无新增地址入口

## Proposed Solution

### 核心设计原则

1. **KISS (Keep It Simple, Stupid)**
   - 在成功状态卡片底部添加"添加地址"按钮
   - 弹出简单对话框选择代币和网络
   - 调用现有后端API

2. **高内聚，低耦合**
   - 新增地址逻辑独立为Dialog组件
   - 复用现有Select组件和网络选项
   - 不影响现有钱包创建流程

3. **100% 测试覆盖率**
   - 所有新增代码路径都有测试

### 实现方案

#### 1. 新增类型定义

```typescript
// 可用网络选项（从Deposit.tsx复用）
const NETWORK_OPTIONS = [
  { value: "TRON", label: "TRON (TRC20)", name: "TRON" },
  { value: "ETH", label: "Ethereum (ERC20)", name: "Ethereum" },
  { value: "BSC", label: "BNB Smart Chain (BEP20)", name: "BNB Smart Chain" },
];

// 代币选项
const TOKEN_OPTIONS = [
  { value: "USDT", label: "USDT" },
  { value: "USDC", label: "USDC" },
];

// 添加地址请求类型
interface AddAddressRequest {
  chain: string;  // TRON, ETH, BSC
  token: string;  // USDT, USDC
}
```

#### 2. 新增状态

```typescript
const [showAddAddressDialog, setShowAddAddressDialog] = useState(false);
const [selectedToken, setSelectedToken] = useState<string>("USDT");
const [selectedChain, setSelectedChain] = useState<string>("TRON");
```

#### 3. 新增Mutation

```typescript
const addAddressMutation = useMutation({
  mutationFn: async (data: AddAddressRequest) => {
    return apiRequest("/api/wallet/addresses", {
      method: "POST",
      headers: { Authorization: `Bearer ${token}` },
      body: JSON.stringify(data),
    });
  },
  onSuccess: () => {
    queryClient.invalidateQueries({ queryKey: ["walletInfo"] });
    toast({ title: "地址添加成功" });
    setShowAddAddressDialog(false);
  },
  onError: (err) => {
    toast({ variant: "destructive", title: "地址添加失败", description: err.message });
  },
});
```

#### 4. UI变更

在成功状态卡片底部添加：

```tsx
{status === "SUCCESS" && (
  <>
    {/* 现有地址显示 */}

    {/* 添加地址按钮 */}
    <Button
      variant="outline"
      onClick={() => setShowAddAddressDialog(true)}
      className="w-full"
    >
      <Plus className="w-4 h-4 mr-2" />
      添加地址
    </Button>

    {/* 添加地址对话框 */}
    <Dialog open={showAddAddressDialog} onOpenChange={setShowAddAddressDialog}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>添加新地址</DialogTitle>
        </DialogHeader>
        <div className="space-y-4">
          {/* 代币选择 */}
          <Select value={selectedToken} onValueChange={setSelectedToken}>
            <SelectTrigger><SelectValue /></SelectTrigger>
            <SelectContent>
              {TOKEN_OPTIONS.map(opt => (
                <SelectItem key={opt.value} value={opt.value}>{opt.label}</SelectItem>
              ))}
            </SelectContent>
          </Select>
          {/* 网络选择 */}
          <Select value={selectedChain} onValueChange={setSelectedChain}>
            <SelectTrigger><SelectValue /></SelectTrigger>
            <SelectContent>
              {NETWORK_OPTIONS.map(opt => (
                <SelectItem key={opt.value} value={opt.value}>{opt.name}</SelectItem>
              ))}
            </SelectContent>
          </Select>
          {/* 确认按钮 */}
          <Button onClick={() => addAddressMutation.mutate({ chain: selectedChain, token: selectedToken })}>
            确认添加
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  </>
)}
```

### 文件变更

1. **`src/pages/dashboard/AccountOpening.tsx`**
   - 添加代币和网络选项常量
   - 添加Dialog和Select组件
   - 添加addAddressMutation
   - 添加状态管理
   - 添加UI组件

2. **`src/pages/dashboard/AccountOpening.test.tsx`**
   - 添加添加地址按钮渲染测试
   - 添加对话框打开/关闭测试
   - 添加代币/网络选择测试
   - 添加API调用测试

### 测试用例

1. `should show add address button when wallet created`
   - 成功状态显示"添加地址"按钮

2. `should open add address dialog when button clicked`
   - 点击按钮打开对话框

3. `should render token selection dropdown`
   - 显示USDT/USDC选项

4. `should render chain selection dropdown`
   - 显示TRON/ETH/BSC选项

5. `should call API when confirm button clicked`
   - 确认后调用正确API

6. `should close dialog after successful add`
   - 成功后关闭对话框

### 风险评估

- **风险级别**: 低
- **影响范围**: 仅AccountOpening页面成功状态
- **回滚方案**: 简单移除新增代码即可

### 验证标准

- [ ] 代码符合KISS原则
- [ ] 高内聚低耦合设计
- [ ] 100%测试覆盖率
- [ ] 无回归影响
- [ ] 构建成功
- [ ] Lint通过
- [ ] 所有测试通过
