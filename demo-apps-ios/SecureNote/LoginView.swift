import SwiftUI

struct LoginView: View {
    @EnvironmentObject var authManager: AuthManager

    @State private var email = ""
    @State private var password = ""
    @State private var isLoading = false

    var body: some View {
        ZStack {
            Color(.systemBackground)
                .ignoresSafeArea()

            VStack(spacing: 20) {
                VStack(spacing: 10) {
                    Image(systemName: "lock.doc")
                        .font(.system(size: 48))
                        .foregroundColor(.blue)

                    Text("SecureNote")
                        .font(.title)
                        .fontWeight(.bold)

                    Text("ネイティブログイン")
                        .font(.subheadline)
                        .foregroundColor(.gray)
                }
                .padding(.bottom, 30)

                VStack(spacing: 12) {
                    TextField("メールアドレス", text: $email)
                        .textContentType(.emailAddress)
                        .keyboardType(.emailAddress)
                        .padding()
                        .background(Color(.systemGray6))
                        .cornerRadius(8)

                    SecureField("パスワード", text: $password)
                        .textContentType(.password)
                        .padding()
                        .background(Color(.systemGray6))
                        .cornerRadius(8)
                }

                if !authManager.errorMessage.isEmpty {
                    Text(authManager.errorMessage)
                        .font(.caption)
                        .foregroundColor(.red)
                        .multilineTextAlignment(.center)
                }

                Button(action: loginTapped) {
                    if isLoading {
                        ProgressView()
                            .tint(.white)
                    } else {
                        Text("ログイン")
                            .fontWeight(.semibold)
                    }
                }
                .frame(maxWidth: .infinity)
                .frame(height: 50)
                .background(Color.blue)
                .foregroundColor(.white)
                .cornerRadius(8)
                .disabled(isLoading || email.isEmpty || password.isEmpty)

                Spacer()

                VStack(spacing: 8) {
                    Text("デモ用ログイン情報:")
                        .font(.caption)
                        .foregroundColor(.gray)

                    HStack(spacing: 16) {
                        VStack(alignment: .leading, spacing: 4) {
                            Text("メール")
                                .font(.caption2)
                                .foregroundColor(.gray)
                            Text("demo@example.com")
                                .font(.caption)
                                .monospaced()
                        }

                        Divider()

                        VStack(alignment: .leading, spacing: 4) {
                            Text("パスワード")
                                .font(.caption2)
                                .foregroundColor(.gray)
                            Text("demo123")
                                .font(.caption)
                                .monospaced()
                        }
                    }
                    .padding()
                    .background(Color(.systemGray6))
                    .cornerRadius(8)
                }
            }
            .padding(20)
        }
    }

    private func loginTapped() {
        isLoading = true

        Task {
            await authManager.login(email: email, password: password)
            isLoading = false
        }
    }
}

#Preview {
    LoginView()
        .environmentObject(AuthManager())
}
