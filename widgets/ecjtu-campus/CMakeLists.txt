cmake_minimum_required(VERSION 3.22)

project(ecjtu-campus-widget VERSION 0.1.0 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_AUTOMOC ON)

include(GNUInstallDirs)
include(CTest)

find_package(Qt6 REQUIRED COMPONENTS Core Gui Qml Quick ShaderTools Test)
find_package(KF6WindowSystem QUIET)
qt_policy(SET QTP0001 NEW)

# 1. C++ QML 模块：io.github.mochenya.ecjtucampus.backend
set(ECJTU_QML_ROOT "${CMAKE_BINARY_DIR}/qml")
set(ECJTU_BACKEND_OUT "${ECJTU_QML_ROOT}/io/github/mochenya/ecjtucampus/backend")

qt_add_qml_module(ecjtucampus_backend
    SHARED
    PLUGIN_TARGET ecjtucampus_backend
    URI io.github.mochenya.ecjtucampus.backend
    VERSION 1.0
    OUTPUT_DIRECTORY "${ECJTU_BACKEND_OUT}"
    SOURCES
        plugin/CourseBackend.h
        plugin/CourseBackend.cpp
        plugin/GlassWindowHelper.h
        plugin/GlassWindowHelper.cpp
)

target_include_directories(ecjtucampus_backend
    PRIVATE
        "${CMAKE_CURRENT_SOURCE_DIR}/plugin"
)

target_link_libraries(ecjtucampus_backend
    PRIVATE
        Qt6::Core
        Qt6::Gui
        Qt6::Qml
)

if(KF6WindowSystem_FOUND)
    target_link_libraries(ecjtucampus_backend PRIVATE KF6::WindowSystem)
    target_compile_definitions(ecjtucampus_backend PRIVATE HAVE_KF6_WINDOWSYSTEM=1)
endif()

# 2. 编译自研 GLSL 440 着色器为 Qt 6 .qsb，并装配自包含 Plasmoid 构建包
set(ECJTU_STAGED_PACKAGE "${CMAKE_BINARY_DIR}/package")
set(ECJTU_STAGED_SHADERS "${ECJTU_STAGED_PACKAGE}/contents/ui/shaders")

set(ECJTU_SHADER_SOURCES
    "${CMAKE_CURRENT_SOURCE_DIR}/shaders/kawase_blur.frag"
    "${CMAKE_CURRENT_SOURCE_DIR}/shaders/liquid_surface.frag"
    "${CMAKE_CURRENT_SOURCE_DIR}/shaders/liquid_pill.frag"
)

set(ECJTU_SHADER_OUTPUTS)
foreach(SHADER_SRC ${ECJTU_SHADER_SOURCES})
    get_filename_component(SHADER_NAME "${SHADER_SRC}" NAME)
    set(SHADER_OUT "${ECJTU_STAGED_SHADERS}/${SHADER_NAME}.qsb")
    add_custom_command(
        OUTPUT "${SHADER_OUT}"
        COMMAND "${CMAKE_COMMAND}" -E make_directory "${ECJTU_STAGED_SHADERS}"
        COMMAND $<TARGET_FILE:Qt6::qsb> --glsl "100es,120,150" -o "${SHADER_OUT}" "${SHADER_SRC}"
        DEPENDS "${SHADER_SRC}"
        COMMENT "Compiling Qt 6 shader ${SHADER_NAME}"
        VERBATIM
    )
    list(APPEND ECJTU_SHADER_OUTPUTS "${SHADER_OUT}")
endforeach()

add_custom_target(ecjtucampus_package ALL
    COMMAND "${CMAKE_COMMAND}" -E copy_directory
            "${CMAKE_CURRENT_SOURCE_DIR}/package"
            "${ECJTU_STAGED_PACKAGE}"
    DEPENDS ${ECJTU_SHADER_OUTPUTS}
    COMMENT "Staging Plasma 6 widget package"
)

# 3. Qt Test 单元测试（仅使用离线脱敏模拟 CLI）
if(BUILD_TESTING)
    add_executable(test_course_backend
        tests/test_course_backend.cpp
        plugin/CourseBackend.h
        plugin/CourseBackend.cpp
        plugin/GlassWindowHelper.h
        plugin/GlassWindowHelper.cpp
    )
    target_include_directories(test_course_backend PRIVATE "${CMAKE_CURRENT_SOURCE_DIR}/plugin")
    target_link_libraries(test_course_backend
        PRIVATE
            Qt6::Core
            Qt6::Gui
            Qt6::Qml
            Qt6::Test
    )
    if(KF6WindowSystem_FOUND)
        target_link_libraries(test_course_backend PRIVATE KF6::WindowSystem)
        target_compile_definitions(test_course_backend PRIVATE HAVE_KF6_WINDOWSYSTEM=1)
    endif()
    target_compile_definitions(test_course_backend
        PRIVATE
            ECJTU_TEST_DIR="${CMAKE_CURRENT_SOURCE_DIR}/tests"
    )
    add_test(NAME test_course_backend COMMAND test_course_backend)
endif()

# 4. 可选用户级安装规则（不自动执行；使用时传入 --prefix ~/.local）
install(
    DIRECTORY "${ECJTU_STAGED_PACKAGE}/"
    DESTINATION "${CMAKE_INSTALL_DATADIR}/plasma/plasmoids/io.github.mochenya.ecjtucampus"
)
install(
    DIRECTORY "${ECJTU_BACKEND_OUT}/"
    DESTINATION "${CMAKE_INSTALL_LIBDIR}/qml/io/github/mochenya/ecjtucampus/backend"
)
