import SwiftUI

struct MainView: View {
    @EnvironmentObject var authManager: AuthManager

    var body: some View {
        ZStack {
            WebViewContainer(authToken: authManager.authToken)

            VStack {
                HStack {
                    VStack(alignment: .leading) {
                        Text("ようこそ")
                            .font(.caption)
                            .foregroundColor(.gray)
                        Text("SecureNote")
                            .font(.headline)
                    }

                    Spacer()

                    Menu {
                        Button(action: logoutTapped) {
                            Label("ログアウト", systemImage: "rectangle.portrait.and.arrow.right")
                        }
                    } label: {
                        Image(systemName: "ellipsis.circle.fill")
                            .font(.title3)
                            .foregroundColor(.blue)
                    }
                }
                .padding()
                .background(Color(.systemBackground))
                .shadow(radius: 1)

                Spacer()
            }
        }
    }

    private func logoutTapped() {
        authManager.logout()
    }
}

#Preview {
    MainView()
        .environmentObject(AuthManager())
}
