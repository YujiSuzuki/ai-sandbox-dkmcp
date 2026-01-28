import Foundation

@Observable
class AuthManager {
    @MainActor
    var isLoggedIn = false

    @MainActor
    var authToken = ""

    @MainActor
    var errorMessage = ""

    private let apiURL = "http://localhost:8080/api"

    @MainActor
    func login(email: String, password: String) async {
        errorMessage = ""

        let loginRequest = LoginRequest(email: email, password: password)

        guard let jsonData = try? JSONEncoder().encode(loginRequest) else {
            errorMessage = "リクエスト作成エラー"
            return
        }

        var request = URLRequest(url: URL(string: "\(apiURL)/auth/login")!)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = jsonData

        do {
            let (data, response) = try await URLSession.shared.data(for: request)

            guard let httpResponse = response as? HTTPURLResponse else {
                errorMessage = "無効なレスポンス"
                return
            }

            if httpResponse.statusCode == 200 {
                let loginResponse = try JSONDecoder().decode(LoginResponse.self, from: data)
                authToken = loginResponse.token
                isLoggedIn = true
            } else {
                let error = try JSONDecoder().decode(ErrorResponse.self, from: data)
                errorMessage = error.message
            }
        } catch {
            errorMessage = "ログインエラー: \(error.localizedDescription)"
        }
    }

    @MainActor
    func logout() {
        authToken = ""
        isLoggedIn = false
    }
}

// MARK: - Models

struct LoginRequest: Codable {
    let email: String
    let password: String
}

struct LoginResponse: Codable {
    let token: String
    let user: User
}

struct User: Codable {
    let id: String
    let email: String
    let name: String
}

struct ErrorResponse: Codable {
    let message: String
}
