// Secrets.swift - Hardcoded Secrets (DEMO SECRET FILE)
// このファイルは秘匿情報のデモ用サンプルです
// 本番環境では絶対にハードコードしないでください！
// 実際には、Secrets.swift を廃止し、xcconfig や環境変数に移行するべきファイルです。

import Foundation

enum Secrets {
    // API Keys
    static let apiKey = "sk-demo-hardcoded-api-key-FAKE"
    static let apiSecret = "demo-secret-value-12345-FAKE"

    // OAuth
    static let googleClientID = "123456789-fake.apps.googleusercontent.com"
    static let googleClientSecret = "GOCSPX-fake-client-secret"

    // Encryption
    static let encryptionKey = "AES256-FAKE-ENCRYPTION-KEY-32BYTES!"
    static let jwtSecret = "jwt-signing-secret-fake-demo-only"

    // Third Party Services
    static let stripePublishableKey = "pk_test_FAKE_stripe_key"
    static let stripeSecretKey = "sk_test_FAKE_stripe_secret"

    // Analytics
    static let mixpanelToken = "fake-mixpanel-token-12345"
    static let amplitudeApiKey = "fake-amplitude-key-67890"
}
