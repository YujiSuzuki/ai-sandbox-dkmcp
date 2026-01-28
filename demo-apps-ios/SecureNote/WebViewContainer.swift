import SwiftUI
import WebKit

struct WebViewContainer: View {
    let authToken: String

    var body: some View {
        ZStack {
            WebView(token: authToken)
                .ignoresSafeArea(.container, axes: .vertical)

            VStack {
                HStack {
                    Text("SecureNote Web")
                        .font(.headline)
                    Spacer()
                }
                .padding()
                .background(Color(.systemBackground))

                Spacer()
            }
        }
    }
}

struct WebView: UIViewRepresentable {
    let token: String
    let webView = WKWebView(frame: .zero, configuration: WKWebViewConfiguration())

    func makeUIView(context: Context) -> WKWebView {
        configureWebView()
        loadWebApp()
        return webView
    }

    func updateUIView(_ webView: WKWebView, context: Context) {}

    private func configureWebView() {
        // JavaScriptの実行を許可
        webView.configuration.preferences.javaScriptEnabled = true

        // ローカルストレージ有効化
        if #available(iOS 14.0, *) {
            webView.configuration.preferences.setValue(true, forKey: "allowFileAccessFromFileURLs")
        }
    }

    private func loadWebApp() {
        // トークンを JavaScript で利用可能にするスクリプトを作成
        let tokenScript = """
        window.authToken = '\(token)';
        localStorage.setItem('auth_token', '\(token)');
        console.log('Auth token injected');
        """

        let userScript = WKUserScript(
            source: tokenScript,
            injectionTime: .atDocumentStart,
            forMainFrameOnly: true
        )

        webView.configuration.userContentController.addUserScript(userScript)

        // localhost:3000 にアクセス
        if let url = URL(string: "http://localhost:3000") {
            let request = URLRequest(url: url)
            webView.load(request)
        }
    }
}

#Preview {
    WebViewContainer(authToken: "demo_token_12345")
}
