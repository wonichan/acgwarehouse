import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'providers/image_provider.dart';
import 'providers/tag_provider.dart';
import 'providers/duplicate_provider.dart';
import 'providers/search_provider.dart';
import 'services/api_service.dart';
import 'services/tag_service.dart';
import 'services/duplicate_service.dart';
import 'services/search_service.dart';
import 'screens/gallery_screen.dart';
import 'screens/duplicate_screen.dart';
import 'screens/search_screen.dart';

void main() {
  runApp(const MyApp());
}

class MyApp extends StatelessWidget {
  const MyApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MultiProvider(
      providers: [
        Provider(create: (_) => ApiService()),
        Provider(create: (_) => TagService()),
        Provider(create: (_) => DuplicateService()),
        Provider(create: (_) => SearchService()),
        ChangeNotifierProvider(create: (context) => ImageListProvider(context.read<ApiService>())..loadImages()),
        ChangeNotifierProvider(create: (context) => TagProvider(context.read<TagService>())),
        ChangeNotifierProvider(create: (context) => DuplicateProvider(service: context.read<DuplicateService>())),
        ChangeNotifierProvider(create: (context) => SearchProvider(service: context.read<SearchService>())),
      ],
      child: MaterialApp(
        title: 'ACGWarehouse',
        theme: ThemeData(
          colorScheme: ColorScheme.fromSeed(seedColor: Colors.blue),
          useMaterial3: true,
        ),
        home: const MainScreen(),
      ),
    );
  }
}

class MainScreen extends StatefulWidget {
  const MainScreen({super.key});

  @override
  State<MainScreen> createState() => _MainScreenState();
}

class _MainScreenState extends State<MainScreen> {
  int _selectedIndex = 0;

  static const List<Widget> _screens = [
    GalleryScreen(),
    SearchScreen(),
    DuplicateScreen(),
  ];

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: _screens[_selectedIndex],
      bottomNavigationBar: NavigationBar(
        selectedIndex: _selectedIndex,
        onDestinationSelected: (index) {
          setState(() {
            _selectedIndex = index;
          });
        },
        destinations: const [
          NavigationDestination(
            icon: Icon(Icons.photo_library_outlined),
            selectedIcon: Icon(Icons.photo_library),
            label: '图库',
          ),
          NavigationDestination(
            icon: Icon(Icons.search_outlined),
            selectedIcon: Icon(Icons.search),
            label: '搜索',
          ),
          NavigationDestination(
            icon: Icon(Icons.content_copy_outlined),
            selectedIcon: Icon(Icons.content_copy),
            label: '重复检测',
          ),
        ],
      ),
    );
  }
}