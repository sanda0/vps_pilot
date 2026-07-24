#include "vps_pilot/project_scanner.hpp"

#include <fstream>
#include <iostream>
#include <system_error>

namespace vps_pilot {

ProjectScanner::ProjectScanner(std::vector<std::filesystem::path> roots)
    : roots_(std::move(roots)) {}

nlohmann::json ProjectScanner::scan() const {
    nlohmann::json projects = nlohmann::json::array();

    for (const auto& root : roots_) {
        std::error_code error;
        if (!std::filesystem::is_directory(root, error)) {
            continue;
        }

        const auto options = std::filesystem::directory_options::skip_permission_denied;
        std::filesystem::recursive_directory_iterator iterator(root, options, error);
        const std::filesystem::recursive_directory_iterator end;
        while (iterator != end) {
            if (error) {
                std::cerr << "project scan warning: " << error.message() << '\n';
                error.clear();
                iterator.increment(error);
                continue;
            }

            if (iterator.depth() > 8) {
                iterator.disable_recursion_pending();
            }

            const auto& entry = *iterator;
            if (entry.is_regular_file(error) && entry.path().filename() == "config.vpspilot.json") {
                try {
                    std::ifstream input(entry.path());
                    auto project = nlohmann::json::parse(input);
                    project["path"] = entry.path().parent_path().string();
                    projects.push_back(std::move(project));
                } catch (const std::exception& exception) {
                    std::cerr << "invalid project config " << entry.path() << ": "
                              << exception.what() << '\n';
                }
            }
            iterator.increment(error);
        }
    }

    return {{"projects", std::move(projects)}};
}

}  // namespace vps_pilot

