cmake_minimum_required(VERSION 3.23)
project(vps_pilot_agent VERSION 0.1.0 LANGUAGES CXX)

find_package(asio CONFIG REQUIRED)
find_package(nlohmann_json CONFIG REQUIRED)

add_executable(vps-pilot-agent
    src/main.cpp
    src/agent.cpp
    src/config.cpp
    src/linux_metrics.cpp
)

target_compile_features(vps-pilot-agent PRIVATE cxx_std_20)
target_include_directories(vps-pilot-agent PRIVATE include)
target_link_libraries(vps-pilot-agent PRIVATE asio::asio nlohmann_json::nlohmann_json)
target_compile_definitions(vps-pilot-agent PRIVATE ASIO_STANDALONE)

if(CMAKE_CXX_COMPILER_ID MATCHES "GNU|Clang")
    target_compile_options(vps-pilot-agent PRIVATE -Wall -Wextra -Wpedantic -Wconversion)
endif()

include(GNUInstallDirs)
install(TARGETS vps-pilot-agent RUNTIME DESTINATION ${CMAKE_INSTALL_BINDIR})
install(FILES config.example.json DESTINATION ${CMAKE_INSTALL_SYSCONFDIR}/vps-pilot-agent)
