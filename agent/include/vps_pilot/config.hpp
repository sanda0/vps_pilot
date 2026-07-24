#pragma once

#include <cstdint>
#include <filesystem>
#include <string>
#include <vector>

namespace vps_pilot {

struct Config {
    std::string server_host{"127.0.0.1"};
    std::uint16_t server_port{55001};
    std::string agent_token;
    std::uint32_t metrics_interval_seconds{5};
    std::uint32_t projects_interval_seconds{300};
    std::vector<std::filesystem::path> project_roots{"/var/www", "/opt"};

    static Config load(const std::filesystem::path& path);
};

}  // namespace vps_pilot
