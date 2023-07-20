# _Build_a_blockchain





## Section3 钱包

钱包是一个结构体Wallet, 包含了用户的公钥和私钥, 我的区块链中使用secp256k1曲线生成公私钥对; 
下面介绍公私钥的功能：
1. 私钥用于签署交易，保证交易的非伪造性
2. 公钥用于验证交易，保证交易的真实性

除了生成公钥和私钥, 还需要有一个函数根据公钥生成一个bitcoin类型的地址, 因此我们需要有一个GetAddressWithPublickey函数, 专门用于生成地址。


### 根据公钥生成地址

根据公钥生成地址需要4个步骤：
1. 对公钥先执行 SHA256 哈希，再执行 RIPEMD160 哈希(需要注意的是，RIPE160算法安全性较低，这里使用它只是为了和bitcoin保持一致)
2. 在公钥hash开头添加一个版本号，构造versionedPayload；
3. 对构造versionedPayload计算校验和并且添加到末尾形成fullPayload：对上述结果连续进行两次sha256并且取出前面4byte
    1. 校验和需要对versionedPayload 进行两次的hash操作并且取前4个byte
4. 对fullPayload进行base58编码，形成bitcoin公钥地址


