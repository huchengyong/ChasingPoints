import XCTest
@testable import iOS

/// 应用内协议正文的结构与关键条款。
final class AgreementDocumentTests: XCTestCase {
    func testUserAgreementMatchesPageStructure() {
        let document = AgreementContent.userAgreement

        XCTAssertEqual(document.navigationTitle, "用户协议")
        XCTAssertEqual(document.title, "追分用户服务协议")
        XCTAssertEqual(document.updatedAt, "更新日期：2024年12月1日")
        XCTAssertEqual(document.sections.count, 8)
        XCTAssertEqual(document.sections.first?.title, "一、总则")
        XCTAssertTrue(document.sections.allSatisfy { !$0.paragraphs.isEmpty })
    }

    func testPrivacyPolicyMatchesPageStructure() {
        let document = AgreementContent.privacyPolicy

        XCTAssertEqual(document.navigationTitle, "隐私政策")
        XCTAssertEqual(document.title, "追分隐私政策")
        XCTAssertEqual(document.updatedAt, "更新日期：2024年12月1日")
        XCTAssertFalse(document.sections.isEmpty)
        XCTAssertTrue(document.sections.allSatisfy { !$0.title.isEmpty && !$0.paragraphs.isEmpty })
    }

    func testEveryParagraphIsNonEmpty() {
        for document in [AgreementContent.userAgreement, AgreementContent.privacyPolicy] {
            for section in document.sections {
                XCTAssertTrue(section.paragraphs.allSatisfy { !$0.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty })
            }
        }
    }
}
