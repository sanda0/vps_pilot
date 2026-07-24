#pragma once

#include <cstdint>
#include <string>
#include <filesystem>

namespace vps_pilot {

struct Config {
    std::string server_host{"127.0.0.1"};
    std::uint16_t server_port{55001};
    std::string agent_token;
    std::uint32_t metrics_interval_seconds{5};

    static Config load(const std::filesystem::path& path);
};

}  // namespace vps_pilot
