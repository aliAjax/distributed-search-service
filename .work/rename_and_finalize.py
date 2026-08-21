from pathlib import Path
import json

ROOT=Path('/Users/hutu/Desktop/go_projects/work_projects/013-distributed-search-service'); DATE='2026-08-21'; REPO='distributed-search-service'
names={
1:['TestDssR01Missing','TestDssR01Conflict','TestDssR01Version','TestDssR01Page'],
2:['TestDssR02Replay','TestDssR02Append','TestDssR02Recover','TestDssR02Copy'],
3:['TestDssR03View','TestDssR03Cache','TestDssR03Catalog','TestDssR03Stats'],
4:['TestDssR04Tokens','TestDssR04Synonyms','TestDssR04Stops','TestDssR04Registry'],
5:['TestDssR05Results','TestDssR05Wait','TestDssR05Errors','TestDssR05Consumer'],
6:['TestDssR06Leases','TestDssR06Primary','TestDssR06Rollback','TestDssR06Close'],
7:['TestDssR07Terminal','TestDssR07Edges','TestDssR07Pending','TestDssR07Audit'],
8:['TestDssR08Options','TestDssR08TypedNil','TestDssR08Facets','TestDssR08Validators'],
9:['TestDssR09Cancel','TestDssR09Deadline','TestDssR09Fresh','TestDssR09Logging'],
10:['TestDssR10Open','TestDssR10Scan','TestDssR10Validate','TestDssR10Classify'],
}
names={record:[f'{name}Z{record:02d}' for name in tests] for record,tests in names.items()}
criteria={
1:'1. 缺失集合包装后仍能识别 missing 哨兵；2. 创建冲突、版本冲突和分页错误分别保持各自错误身份；3. 四条定向检查连续 10 轮红绿结果一致，撤掉任一包装修复就重新失败；4. 仓储及全量回归测试全部通过。',
2:'1. 已取消的 WAL 回放不会调用应用函数；2. 追加、恢复和对象复制均尊重原请求 deadline；3. 对四种取消场景各跑 10 轮都稳定，撤销任何一处上下文传递后该项立刻恢复红灯；4. 存储包与全量回归无新增失败。',
3:'1. 索引快照被调用方改写后内部首项仍保持 base；2. 缓存、段目录和代际统计在并发读写下没有 data race；3. 四条竞态命令连续 10 轮通过，撤回任一锁或复制修复便重新变红；4. 全量回归在竞态修复后保持通过。',
4:'1. 词元过滤不改写调用方传入切片；2. 同义词、停用词和注册表快照返回值均不共享底层数组；3. 四种组合调用连续 10 轮内容一致，去掉任一复制修复后对应断言恢复失败；4. 分析器与全量回归测试全部正常。',
5:'1. 分片报错后结果通道及时关闭且消费协程退出；2. WaitGroup 必须等待全部分片，错误发送方不得因无人接收而泄漏；3. 四条并发检查连续 10 轮无挂起，撤回任一生命周期修复就重新变红；4. 分片包和全量回归测试通过。',
6:'1. 每次压缩租约在下一项开始前释放；2. 操作错误不会被 cleanup 或 close 错误覆盖，回滚也确实执行；3. 四个资源分支连续 10 轮稳定，移除任一 defer 修复后对应目标恢复失败；4. 维护模块及全量回归保持绿色。',
7:'1. 先稳定复现重试成功后仍停在 retrying；2. 查明状态转换表、终态写回、进行中视图和审计值的具体文件与符号；3. 解释四处状态错位如何让查询和审计互相矛盾；4. 全程只读排查，不得改代码且不能产生代码差异。',
8:'1. 实际触发零值 Options 和 Facets 写入时的 panic；2. 报出 typed nil 策略及校验器集合对应源码位置和函数名；3. 讲清四条零值路径为何分别崩溃或绕过校验；4. 只允许读取和运行复现，不得改代码，结束时源码无差异。',
9:'1. 复现客户端取消后下游仍被调用以及 deadline 消失；2. 找出请求上下文被替换、复用和日志消费涉及的文件与符号；3. 说明连续请求间旧上下文污染的传播过程；4. 排查期间不得写源码，所有文件必须保持零代码改动。',
10:'1. 分别复现配置打开、扫描、校验和启动分类无法识别哨兵错误；2. 定位四个包装边界的文件与符号；3. 说明错误链断开后监控归类失败的完整路径；4. 仅做诊断不得改代码，结束时工作区不能产生代码差异。',
}
plan_path=ROOT/'_shared/bug_plan.json'; plan=json.loads(plan_path.read_text())
for b in plan['bugs']:
    n=int(b['record']); project=ROOT/DATE/f'{REPO}__{n:03d}'; data=json.loads((project/'collection.json').read_text())
    old=b['tests']; new=names[n]
    for base in (project/'env', ROOT/'_gold'/f'{REPO}__{n:03d}'):
        for p in base.rglob('*_test.go'):
            text=p.read_text()
            for a,z in zip(old,new): text=text.replace(a,z)
            p.write_text(text)
    commands=[]
    for command,a,z in zip(b['verify_cmds'].splitlines(),old,new): commands.append(command.replace(a,z))
    b['tests']=new; b['verify_cmds']='\n'.join(commands); b['success_criteria']=criteria[n]
    if n==5: b['has_stack']=True
    data['verify_cmds']=b['verify_cmds']; data['success_criteria']=criteria[n]
    data['user_query']=(project/'prompt.txt').read_text().strip()
    for claim,z,command in zip(data['coverage_contract']['claims'],new,commands): claim['test']=z; claim['command']=command
    (project/'collection.json').write_text(json.dumps(data,ensure_ascii=False,indent=2)+'\n')
plan_path.write_text(json.dumps(plan,ensure_ascii=False,indent=2)+'\n')
