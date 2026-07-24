#include "vps_pilot/config.hpp"

#include <fstream>
#include <limits>
#include <stdexcept>

#include <nlohmann/json.hpp>

namespace vps_pilot {

Config Config::load(const std::filesystem::path& path) {
    std::ifstream input(path);
    if (!input) {
        throw std::runtime_error("cannot open configuration file: " + path.string());
    }

    const auto json = nlohmann::json::parse(input);
    Config config;
    config.server_host = json.value("server_host", config.server_host);
    config.agent_token = json.value("agent_token", config.agent_token);

    const auto port = json.value("server_port", static_cast<unsigned>(config.server_port));
    if (port == 0 || port > std::numeric_limits<std::uint16_t>::max()) {
        throw std::runtime_error("server_port must be between 1 and 65535");
    }
    config.server_port = static_cast<std::uint16_t>(port);
    config.metrics_interval_seconds =
        json.value("metrics_interval_seconds", config.metrics_interval_seconds);
    if (config.metrics_interval_seconds == 0) {
        throw std::runtime_error("metrics_interval_seconds must be greater than zero");
    }
    return config;
}

}  // namespace vps_pilot
