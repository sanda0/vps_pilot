#pragma once

#include <filesystem>
#include <vector>

#include <nlohmann/json.hpp>

namespace vps_pilot {

class ProjectScanner {
public:
    explicit ProjectScanner(std::vector<std::filesystem::path> roots);
    nlohmann::json scan() const;

private:
    std::vector<std::filesystem::path> roots_;
};

}  // namespace vps_pilot

